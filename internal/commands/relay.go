package commands

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/weeb-vip/user-service/config"
	"github.com/weeb-vip/user-service/handlers"
	"github.com/weeb-vip/user-service/internal/logger"
	"github.com/weeb-vip/user-service/tracing"
)

func configureRelayCommand(rootCmd *cobra.Command) {
	relayCmd := &cobra.Command{
		Use:   "relay",
		Short: "move committed events out of the database",
	}

	// Its own command, deployed on its own, so a NATS outage never slows a
	// request: the GraphQL server only ever writes rows.
	outboxCmd := &cobra.Command{
		Use:   "outbox",
		Short: "publish outbox_events rows to NATS JetStream until stopped",
		RunE:  startOutboxRelay,
	}

	relayCmd.AddCommand(outboxCmd)
	rootCmd.AddCommand(relayCmd)
}

func startOutboxRelay(cmd *cobra.Command, args []string) error {
	cfg := config.LoadConfigOrPanic()

	logger.Logger(
		logger.WithServerName("user-service"),
		logger.WithVersion(cfg.APPConfig.Version),
		logger.WithEnvironment(cfg.APPConfig.Env),
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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

	return handlers.OutboxRelayWithContext(tracedCtx, cfg)
}
