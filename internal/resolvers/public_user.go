package resolvers

import (
	"context"
	"time"

	"github.com/weeb-vip/user-service/graph/model"
	"github.com/weeb-vip/user-service/internal/services/users"
	"github.com/weeb-vip/user-service/metrics"
	"github.com/weeb-vip/user-service/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// UserByUsername backs a public user page.
//
// No auth: this is the one query anyone can make about another user, which is
// exactly why it returns PublicUser rather than User -- the safe subset, never
// email or sessions. An unknown username is a null result, not an error: that
// is a 404 for the page above, not a failure worth a span marked bad.
func UserByUsername(ctx context.Context, userService users.User, username string) (*model.PublicUser, error) {
	tracer := tracing.GetTracer(ctx)
	ctx, span := tracer.Start(ctx, "UserByUsername",
		trace.WithAttributes(
			attribute.String("resolver.name", "UserByUsername"),
			attribute.String("username", username),
		),
		tracing.GetEnvironmentAttribute(),
	)
	defer span.End()

	startTime := time.Now()

	user, err := userService.GetUserByUsername(ctx, username)
	if err != nil {
		span.RecordError(err)
		metrics.GetAppMetrics().ResolverMetric(float64(time.Since(startTime).Milliseconds()), "UserByUsername", metrics.Error)
		return nil, err
	}
	if user == nil {
		span.SetAttributes(attribute.Bool("user.found", false))
		metrics.GetAppMetrics().ResolverMetric(float64(time.Since(startTime).Milliseconds()), "UserByUsername", metrics.Success)
		return nil, nil
	}

	metrics.GetAppMetrics().ResolverMetric(float64(time.Since(startTime).Milliseconds()), "UserByUsername", metrics.Success)
	return toPublicUser(user), nil
}
