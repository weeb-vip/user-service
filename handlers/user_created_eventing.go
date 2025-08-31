package handlers

import (
	"context"
	"github.com/ThatCatDev/ep/v2/drivers"
	epKafka "github.com/ThatCatDev/ep/v2/drivers/kafka"
	"github.com/ThatCatDev/ep/v2/event"
	"github.com/ThatCatDev/ep/v2/middlewares/kafka/backoffretry"
	"github.com/ThatCatDev/ep/v2/processor"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/weeb-vip/user/config"
	"github.com/weeb-vip/user/graph/model"
	"github.com/weeb-vip/user/internal/logger"
	"github.com/weeb-vip/user/internal/resolvers"
	"github.com/weeb-vip/user/internal/services/users"
	"go.uber.org/zap"
)

type Payload struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

func UserCreatedEventing() error {
	cfg, _ := config.LoadConfig()
	ctx := context.Background()
	log := logger.Get()
	ctx = logger.WithCtx(ctx, log)

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
			log.Error("Error closing Kafka driver", zap.String("error", err.Error()))
		} else {
			log.Info("Kafka driver closed successfully")
		}
	}(driver)

	processorInstance := processor.NewProcessor[*kafka.Message, Payload](driver, cfg.KafkaConfig.Topic, process)

	log.Info("initializing backoff retry middleware", zap.String("topic", cfg.KafkaConfig.Topic))
	backoffRetryInstance := backoffretry.NewBackoffRetry[Payload](driver, backoffretry.Config{
		MaxRetries: 3,
		HeaderKey:  "retry",
		RetryQueue: cfg.KafkaConfig.Topic + "-retry",
	})

	log.Info("Starting Kafka processor", zap.String("topic", cfg.KafkaConfig.Topic))
	// create middleware to log errors and continue processing

	err := processorInstance.
		AddMiddleware(backoffRetryInstance.Process).
		Run(ctx)

	if err != nil && ctx.Err() == nil { // Ignore error if caused by context cancellation
		log.Error("Error consuming messages", zap.String("error", err.Error()))
		return err
	}

	return nil
}

func process(ctx context.Context, data event.Event[*kafka.Message, Payload]) (event.Event[*kafka.Message, Payload], error) {
	log := logger.FromCtx(ctx)
	if data.Payload.UserID == "" {
		log.Error("Payload is nil")
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
		log.Error("Failed to create user", zap.String("error", err.Error()))
		return data, err
	}

	return data, nil
}
