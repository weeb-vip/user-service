package commands

import (
	"context"

	"github.com/weeb-vip/user-service/config"
	"github.com/weeb-vip/user-service/handlers"
	"github.com/weeb-vip/user-service/internal/logger"
	"github.com/weeb-vip/user-service/tracing"

	"github.com/spf13/cobra"
)

func configureUserCreatedEventCommand(rootCmd *cobra.Command) {
	var eventingCmd = &cobra.Command{
		Use:   "eventing",
		Short: "manipulate eventing",
	}

	var userCreatedStartCmd = &cobra.Command{
		Use:   "user-created",
		Short: "start listening to events",
		RunE:  startUserCreatedEventing,
	}

	// The NATS twin. A separate subcommand rather than a flag: prod and staging
	// run the same image, so the command name is what selects the transport,
	// keeping that choice in the deployment values.
	var userCreatedNatsStartCmd = &cobra.Command{
		Use:   "user-created-nats",
		Short: "start listening to user-created events on NATS JetStream",
		RunE:  startUserCreatedEventingNats,
	}

	rootCmd.AddCommand(eventingCmd)
	eventingCmd.AddCommand(userCreatedStartCmd)
	eventingCmd.AddCommand(userCreatedNatsStartCmd)
}

func startUserCreatedEventing(cmd *cobra.Command, args []string) error {
	// Load config to get environment
	cfg := config.LoadConfigOrPanic()

	// Initialize logger with environment
	logger.Logger(
		logger.WithServerName("user-service"),
		logger.WithVersion("1.0.0"),
		logger.WithEnvironment(cfg.APPConfig.Env),
	)

	// Initialize tracing
	ctx := context.Background()
	tracedCtx, err := tracing.InitTracing(ctx)
	if err != nil {
		log := logger.FromCtx(ctx)
		log.Error().Err(err).Msg("Failed to initialize tracing")
		// Continue without tracing if initialization fails
		tracedCtx = ctx
	} else {
		defer func() {
			if err := tracing.Shutdown(context.Background()); err != nil {
				log := logger.FromCtx(tracedCtx)
				log.Error().Err(err).Msg("Error shutting down tracing")
			}
		}()
		log := logger.FromCtx(tracedCtx)
		log.Info().Msg("Tracing initialized successfully")
	}

	// Start eventing with traced context
	return handlers.UserCreatedEventingWithContext(tracedCtx)
}

// startUserCreatedEventingNats mirrors startUserCreatedEventing, differing only
// in which handler it starts.
func startUserCreatedEventingNats(cmd *cobra.Command, args []string) error {
	cfg := config.LoadConfigOrPanic()

	logger.Logger(
		logger.WithServerName("user-service"),
		logger.WithVersion("1.0.0"),
		logger.WithEnvironment(cfg.APPConfig.Env),
	)

	ctx := context.Background()
	tracedCtx, err := tracing.InitTracing(ctx)
	if err != nil {
		log := logger.FromCtx(ctx)
		log.Error().Err(err).Msg("Failed to initialize tracing")
		tracedCtx = ctx
	} else {
		defer func() {
			if err := tracing.Shutdown(context.Background()); err != nil {
				log := logger.FromCtx(tracedCtx)
				log.Error().Err(err).Msg("Error shutting down tracing")
			}
		}()
	}

	return handlers.UserCreatedEventingNatsWithContext(tracedCtx)
}
