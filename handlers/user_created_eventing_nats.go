package handlers

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/ThatCatDev/ep/v2/drivers"
	epNats "github.com/ThatCatDev/ep/v2/drivers/nats"
	"github.com/ThatCatDev/ep/v2/middlewares/nats/backoffretry"
	"github.com/ThatCatDev/ep/v2/processor"
	"github.com/weeb-vip/user-service/config"
	"github.com/weeb-vip/user-service/internal/logger"
)

// UserCreatedEventingNats is the NATS counterpart of UserCreatedEventing.
//
// A separate entry point rather than a flag, so production keeps running the
// Kafka consumer untouched while staging moves over. It shares process with the
// Kafka handler -- that function is generic over the driver message because it
// only reads Payload -- so the user-creation logic exists once.
//
// No transform middleware, matching the Kafka handler. ep hands the raw body to
// Event.Transform and both drivers pass it the same way, so Payload is
// populated identically. Only the CDC consumers need a transform, because their
// bodies carry a Debezium envelope; user-created is auth's own JSON.
const (
	maxRetries     = 3
	retryHeaderKey = "retry"
)

func UserCreatedEventingNats() error {
	return UserCreatedEventingNatsWithContext(context.Background())
}

func UserCreatedEventingNatsWithContext(ctx context.Context) error {
	cfg, _ := config.LoadConfig()
	log := logger.FromCtx(ctx)

	retrySubject := cfg.NatsConfig.Subject + "-retry"
	dlqSubject := cfg.NatsConfig.Subject + "-dlq"

	// The main consumer: reads the subject auth publishes to, and hands
	// failures to the retry subject.
	driver := epNats.NewNatsDriver(&epNats.Config{
		URL:               cfg.NatsConfig.URL,
		ConsumerGroupName: cfg.NatsConfig.ConsumerGroupName,
		// Empty StreamName: user-created is produced by auth, not Debezium, so
		// the driver creates a stream from the subject rather than binding to
		// the CDC one.
		StreamName:              cfg.NatsConfig.StreamName,
		ConsumerAutoOffsetReset: &cfg.NatsConfig.Offset,
	})
	defer closeDriver(ctx, driver, "NATS driver")

	// The retry consumer, in this same process rather than a second
	// deployment. It needs its own driver because the durable consumer name is
	// driver-level configuration, not per-subject: two Consume calls on one
	// driver would call CreateOrUpdateConsumer with the same durable name and
	// different filter subjects, and the second would reconfigure the first.
	retryDriver := epNats.NewNatsDriver(&epNats.Config{
		URL:                     cfg.NatsConfig.URL,
		ConsumerGroupName:       cfg.NatsConfig.ConsumerGroupName + "-retry",
		StreamName:              cfg.NatsConfig.StreamName,
		ConsumerAutoOffsetReset: &cfg.NatsConfig.Offset,
	})
	defer closeDriver(ctx, retryDriver, "NATS retry driver")

	mainProcessor := processor.NewProcessor[*epNats.Message, Payload](driver, cfg.NatsConfig.Subject, process[*epNats.Message]).
		AddMiddleware(backoffretry.NewBackoffRetry[Payload](driver, backoffretry.Config{
			MaxRetries: maxRetries,
			HeaderKey:  retryHeaderKey,
			RetryQueue: retrySubject,
		}).Process)

	// Exhausted retries go to a dead-letter subject rather than back onto the
	// retry subject. ep acks and drops a message once the counter reaches
	// MaxRetries, so cycling it here would make a permanently failing event
	// disappear with no record -- which is exactly how broken user creation
	// went unnoticed for seventeen days.
	retryProcessor := processor.NewProcessor[*epNats.Message, Payload](retryDriver, retrySubject, process[*epNats.Message]).
		AddMiddleware(backoffretry.NewBackoffRetry[Payload](retryDriver, backoffretry.Config{
			MaxRetries: maxRetries,
			HeaderKey:  retryHeaderKey,
			RetryQueue: dlqSubject,
		}).Process)

	log.Info().
		Str("subject", cfg.NatsConfig.Subject).
		Str("retry_subject", retrySubject).
		Str("dlq_subject", dlqSubject).
		Msg("Starting NATS processors")

	// One consumer returning must stop the other: Consume blocks until its
	// iterator is stopped, so without cancelling here a dead main consumer
	// would leave the process alive and apparently healthy.
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error { return mainProcessor.Run(groupCtx) })
	group.Go(func() error { return retryProcessor.Run(groupCtx) })

	if err := group.Wait(); err != nil && ctx.Err() == nil {
		log.Error().Err(err).Msg("Error consuming messages")

		return err
	}

	return nil
}

func closeDriver(ctx context.Context, driver drivers.Driver[*epNats.Message], name string) {
	log := logger.FromCtx(ctx)
	if err := driver.Close(); err != nil {
		log.Error().Err(err).Msgf("Error closing %s", name)

		return
	}

	log.Info().Msgf("%s closed successfully", name)
}
