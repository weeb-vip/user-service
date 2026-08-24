package handlers

import (
	"context"

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
func UserCreatedEventingNats() error {
	return UserCreatedEventingNatsWithContext(context.Background())
}

func UserCreatedEventingNatsWithContext(ctx context.Context) error {
	cfg, _ := config.LoadConfig()
	log := logger.FromCtx(ctx)

	natsConfig := &epNats.Config{
		URL:               cfg.NatsConfig.URL,
		ConsumerGroupName: cfg.NatsConfig.ConsumerGroupName,
		// Empty StreamName: user-created is produced by auth, not Debezium, so
		// the driver creates a stream from the subject rather than binding to
		// the CDC one.
		StreamName:              cfg.NatsConfig.StreamName,
		ConsumerAutoOffsetReset: &cfg.NatsConfig.Offset,
	}

	driver := epNats.NewNatsDriver(natsConfig)
	defer func(driver drivers.Driver[*epNats.Message]) {
		if err := driver.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing NATS driver")
		} else {
			log.Info().Msg("NATS driver closed successfully")
		}
	}(driver)

	processorInstance := processor.NewProcessor[*epNats.Message, Payload](driver, cfg.NatsConfig.Subject, process[*epNats.Message])

	log.Info().Str("subject", cfg.NatsConfig.Subject).Msg("initializing backoff retry middleware")
	backoffRetryInstance := backoffretry.NewBackoffRetry[Payload](driver, backoffretry.Config{
		MaxRetries: 3,
		HeaderKey:  "retry",
		RetryQueue: cfg.NatsConfig.Subject + "-retry",
	})

	log.Info().Str("subject", cfg.NatsConfig.Subject).Msg("Starting NATS processor")

	err := processorInstance.
		AddMiddleware(backoffRetryInstance.Process).
		Run(ctx)

	if err != nil && ctx.Err() == nil { // Ignore error if caused by context cancellation
		log.Error().Err(err).Msg("Error consuming messages")

		return err
	}

	return nil
}
