package handlers

import (
	"context"
	"time"

	epnats "github.com/ThatCatDev/ep/v2/drivers/nats"
	"github.com/weeb-vip/go-outbox-lib"

	"github.com/weeb-vip/user-service/config"
	"github.com/weeb-vip/user-service/internal/db"
	"github.com/weeb-vip/user-service/internal/logger"
	"github.com/weeb-vip/user-service/metrics"
)

// OutboxRelayWithContext runs the go-outbox-lib relay until ctx is cancelled.
//
// The NATS driver is the same ep driver the consumers use, with no stream
// name: the driver then makes one stream per subject, which is how the
// nats-streams values in weeb-argocd expect to find them.
func OutboxRelayWithContext(ctx context.Context, cfg *config.Config) error {
	log := logger.FromCtx(ctx)

	driver := epnats.NewNatsDriver(&epnats.Config{URL: cfg.NatsConfig.URL})
	defer closeDriver(ctx, driver, "NATS driver")

	relay := outbox.NewRelay(
		db.GetDBService().GetDB(),
		outbox.NewNatsPublisher(driver),
		outbox.Config{
			PollInterval:    time.Duration(cfg.OutboxConfig.PollIntervalMs) * time.Millisecond,
			BatchSize:       cfg.OutboxConfig.BatchSize,
			Retention:       time.Duration(cfg.OutboxConfig.RetentionHours) * time.Hour,
			CleanupInterval: time.Duration(cfg.OutboxConfig.CleanupMinutes) * time.Minute,
			BacklogInterval: time.Duration(cfg.OutboxConfig.BacklogSeconds) * time.Second,
		},
		outbox.WithLogger(log),
		outbox.WithMetrics(outboxMetrics{}),
	)

	log.Info().Str("nats_url", cfg.NatsConfig.URL).Msg("Starting outbox relay")

	return relay.Run(ctx)
}

// outboxMetrics adapts the relay's observations to go-metrics-lib.
type outboxMetrics struct{}

func (outboxMetrics) Published(subject string) {
	_ = metrics.NewMetricsInstance().CountMetric("outbox_published_total", map[string]string{
		"service": "user-service", "subject": subject, "result": metrics.Success, "env": metrics.GetCurrentEnv(),
	})
}

func (outboxMetrics) PublishFailed(subject string) {
	_ = metrics.NewMetricsInstance().CountMetric("outbox_published_total", map[string]string{
		"service": "user-service", "subject": subject, "result": metrics.Error, "env": metrics.GetCurrentEnv(),
	})
}

func (outboxMetrics) Backlog(unpublished int64) {
	_ = metrics.NewMetricsInstance().GaugeMetric("outbox_unpublished", float64(unpublished), map[string]string{
		"service": "user-service", "env": metrics.GetCurrentEnv(),
	})
}
