package middleware

import (
	"context"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/weeb-vip/user-service/metrics"
)

// GraphQLMetricsExtension provides Prometheus metrics for GraphQL operations
type GraphQLMetricsExtension struct{}

// ExtensionName returns the name of the extension
func (e GraphQLMetricsExtension) ExtensionName() string {
	return "GraphQLMetrics"
}

// Validate validates the extension configuration
func (e GraphQLMetricsExtension) Validate(schema graphql.ExecutableSchema) error {
	return nil
}

// InterceptOperation records metrics for GraphQL operations
func (e GraphQLMetricsExtension) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	return next(ctx)
}

// InterceptField records metrics for GraphQL field resolutions
func (e GraphQLMetricsExtension) InterceptField(ctx context.Context, next graphql.Resolver) (interface{}, error) {
	fc := graphql.GetFieldContext(ctx)

	// Only measure root-level resolvers (Query/Mutation/Subscription fields)
	isRootField := fc.Field.ObjectDefinition.Name == "Query" ||
	              fc.Field.ObjectDefinition.Name == "Mutation" ||
	              fc.Field.ObjectDefinition.Name == "Subscription"

	if !isRootField {
		return next(ctx)
	}

	start := time.Now()
	result, err := next(ctx)
	duration := float64(time.Since(start).Nanoseconds()) / float64(time.Millisecond)

	// Record resolver metrics
	appMetrics := metrics.GetAppMetrics()
	resultStatus := "success"
	if err != nil {
		resultStatus = "error"
	}

	resolverName := fc.Field.ObjectDefinition.Name + "." + fc.Field.Name
	appMetrics.ResolverMetric(duration, resolverName, resultStatus)

	return result, err
}