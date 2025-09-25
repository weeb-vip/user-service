package handlers

import (
	"context"
	"github.com/ThatCatDev/ep/v2/drivers"
	epKafka "github.com/ThatCatDev/ep/v2/drivers/kafka"
	"github.com/ThatCatDev/ep/v2/event"
	"github.com/ThatCatDev/ep/v2/middlewares/kafka/backoffretry"
	"github.com/ThatCatDev/ep/v2/processor"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/weeb-vip/user-service/config"
	"github.com/weeb-vip/user-service/graph/model"
	"github.com/weeb-vip/user-service/internal/logger"
	"github.com/weeb-vip/user-service/internal/resolvers"
	"github.com/weeb-vip/user-service/internal/services/users"
)

type Payload struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

func UserCreatedEventing() error {
	return UserCreatedEventingWithContext(context.Background())
}

func UserCreatedEventingWithContext(ctx context.Context) error {
	cfg, _ := config.LoadConfig()
	log := logger.FromCtx(ctx)

	kafkaConfig := &epKafka.KafkaConfig{
		ConsumerGroupName:        cfg.KafkaConfig.ConsumerGroupName,
		BootstrapServers:         cfg.KafkaConfig.BootstrapServers,
		SaslMechanism:            nil,
		SecurityProtocol:         nil,
		Username:                 nil,
		Password:                 nil,
		ConsumerSessionTimeoutMs: nil,
		ConsumerAutoOffsetReset:  &cfg.KafkaConfig.Offset,
		ClientID:                 nil,
		Debug:                    nil,
	}

	driver := epKafka.NewKafkaDriver(kafkaConfig)
	defer func(driver drivers.Driver[*kafka.Message]) {
		err := driver.Close()
		if err != nil {
			log.Error().Err(err).Msg("Error closing Kafka driver")
		} else {
			log.Info().Msg("Kafka driver closed successfully")
		}
	}(driver)

	processorInstance := processor.NewProcessor[*kafka.Message, Payload](driver, cfg.KafkaConfig.Topic, process)

	log.Info().Str("topic", cfg.KafkaConfig.Topic).Msg("initializing backoff retry middleware")
	backoffRetryInstance := backoffretry.NewBackoffRetry[Payload](driver, backoffretry.Config{
		MaxRetries: 3,
		HeaderKey:  "retry",
		RetryQueue: cfg.KafkaConfig.Topic + "-retry",
	})

	log.Info().Str("topic", cfg.KafkaConfig.Topic).Msg("Starting Kafka processor")
	// create middleware to log errors and continue processing

	err := processorInstance.
		AddMiddleware(backoffRetryInstance.Process).
		Run(ctx)

	if err != nil && ctx.Err() == nil { // Ignore error if caused by context cancellation
		log.Error().Err(err).Msg("Error consuming messages")
		return err
	}

	return nil
}

func process(ctx context.Context, data event.Event[*kafka.Message, Payload]) (event.Event[*kafka.Message, Payload], error) {
	log := logger.FromCtx(ctx)
	if data.Payload.UserID == "" {
		log.Error().Msg("Payload is nil")
		// skip, will always fail
		return data, nil
	}
	_, err := resolvers.CreateUserFromEvent(ctx, users.NewUserService(), &model.CreateUserInput{
		ID:        data.Payload.UserID,
		Firstname: "",
		Lastname:  "",
		Username:  "",
		Language:  "EN",
		Email:     &data.Payload.Email,
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to create user")
		return data, err
	}

	return data, nil
}
