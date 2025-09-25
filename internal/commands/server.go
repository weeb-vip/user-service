package commands

import (
	"context"

	"github.com/weeb-vip/user-service"
	"github.com/weeb-vip/user-service/config"
	"github.com/weeb-vip/user-service/internal/logger"
	"github.com/weeb-vip/user-service/tracing"

	"github.com/spf13/cobra"
)

func configureServerCommand(rootCmd *cobra.Command) {
	var serverCmd = &cobra.Command{
		Use:   "server",
		Short: "manipulate server",
	}

	var serverStartCmd = &cobra.Command{
		Use:   "start",
		Short: "start listening to requests",
		RunE:  startServer,
	}

	rootCmd.AddCommand(serverCmd)
	serverCmd.AddCommand(serverStartCmd)
}

func startServer(cmd *cobra.Command, args []string) error {
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

	// Start the server with traced context
	return user.StartServerWithContext(tracedCtx)
}
