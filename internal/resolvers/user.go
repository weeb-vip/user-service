package resolvers

import (
	"context"
	"time"

	"github.com/weeb-vip/user-service/graph/model"
	"github.com/weeb-vip/user-service/http/handlers/requestinfo"
	"github.com/weeb-vip/user-service/internal/logger"
	"github.com/weeb-vip/user-service/internal/services/users"
	"github.com/weeb-vip/user-service/metrics"
	"github.com/weeb-vip/user-service/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func CreateUser( // nolint
	ctx context.Context,
	userService users.User,
	input *model.CreateUserInput,
) (*model.User, error) {
	tracer := tracing.GetTracer(ctx)
	ctx, span := tracer.Start(ctx, "CreateUser",
		trace.WithAttributes(
			attribute.String("user.id", input.ID),
			attribute.String("resolver.name", "CreateUser"),
		),
		tracing.GetEnvironmentAttribute(),
	)
	defer span.End()

	startTime := time.Now()
	log := logger.FromCtx(ctx)
	req := requestinfo.FromContext(ctx)

	userID := req.UserID
	if userID == nil {
		log.Error().Msg("User ID is missing")
		metrics.GetAppMetrics().ResolverMetric(
			float64(time.Since(startTime).Milliseconds()),
			"CreateUser",
			metrics.Error,
		)
		return nil, nil
	}

	if input.ID != *userID {
		log.Error().Msg("User ID does not match, unauthenticated")
		metrics.GetAppMetrics().ResolverMetric(
			float64(time.Since(startTime).Milliseconds()),
			"CreateUser",
			metrics.Error,
		)
		return nil, nil
	}
	language := input.Language.String()
	createdUser, err := userService.AddUser(ctx, input.ID, input.Username, input.Firstname, input.Lastname, language)

	if err != nil {
		metrics.GetAppMetrics().ResolverMetric(
			float64(time.Since(startTime).Milliseconds()),
			"CreateUser",
			metrics.Error,
		)
		return nil, err
	}

	metrics.GetAppMetrics().ResolverMetric(
		float64(time.Since(startTime).Milliseconds()),
		"CreateUser",
		metrics.Success,
	)

	return &model.User{
		ID: createdUser.ID,
	}, nil
}

func CreateUserFromEvent( // nolint
	ctx context.Context,
	userService users.User,
	input *model.CreateUserInput,
) (*model.User, error) {
	tracer := tracing.GetTracer(ctx)
	ctx, span := tracer.Start(ctx, "CreateUserFromEvent",
		trace.WithAttributes(
			attribute.String("user.id", input.ID),
			attribute.String("resolver.name", "CreateUserFromEvent"),
			attribute.String("event.source", "kafka"),
		),
		tracing.GetEnvironmentAttribute(),
	)
	defer span.End()

	startTime := time.Now()

	language := input.Language.String()
	createdUser, err := userService.AddUser(ctx, input.ID, input.Username, input.Firstname, input.Lastname, language)

	if err != nil {
		metrics.GetAppMetrics().ResolverMetric(
			float64(time.Since(startTime).Milliseconds()),
			"CreateUserFromEvent",
			metrics.Error,
		)
		return nil, err
	}

	metrics.GetAppMetrics().ResolverMetric(
		float64(time.Since(startTime).Milliseconds()),
		"CreateUserFromEvent",
		metrics.Success,
	)

	return &model.User{
		ID: createdUser.ID,
	}, nil
}

func GetUser( // nolint
	ctx context.Context,
	userService users.User,
) (*model.User, error) {
	tracer := tracing.GetTracer(ctx)
	ctx, span := tracer.Start(ctx, "GetUser",
		trace.WithAttributes(
			attribute.String("resolver.name", "GetUser"),
		),
		tracing.GetEnvironmentAttribute(),
	)
	defer span.End()

	startTime := time.Now()
	log := logger.FromCtx(ctx)
	req := requestinfo.FromContext(ctx)

	userID := req.UserID
	if userID == nil {
		log.Error().Msg("User ID is missing")
		metrics.GetAppMetrics().ResolverMetric(
			float64(time.Since(startTime).Milliseconds()),
			"GetUser",
			metrics.Error,
		)
		return nil, nil
	}

	// Add user ID to span attributes
	span.SetAttributes(attribute.String("user.id", *userID))

	log.Info().Any("userid", *userID).Msg("Fetching user with ID")
	user, err := userService.GetUserDetails(ctx, *userID)

	if err != nil {
		metrics.GetAppMetrics().ResolverMetric(
			float64(time.Since(startTime).Milliseconds()),
			"GetUser",
			metrics.Error,
		)
		return nil, err
	}

	// convert user language to model.Language
	language := model.Language(user.Language)

	metrics.GetAppMetrics().ResolverMetric(
		float64(time.Since(startTime).Milliseconds()),
		"GetUser",
		metrics.Success,
	)

	return &model.User{
		ID:              user.ID,
		Firstname:       user.FirstName,
		Lastname:        user.LastName,
		Username:        user.Username,
		Language:        language,
		Email:           user.Email,
		ProfileImageURL: user.ProfileImageURL,
	}, nil
}

func UpdateUser( // nolint
	ctx context.Context,
	userService users.User,
	input *model.UpdateUserInput,
) (*model.User, error) {
	tracer := tracing.GetTracer(ctx)
	ctx, span := tracer.Start(ctx, "UpdateUser",
		trace.WithAttributes(
			attribute.String("resolver.name", "UpdateUser"),
		),
		tracing.GetEnvironmentAttribute(),
	)
	defer span.End()

	startTime := time.Now()
	log := logger.FromCtx(ctx)
	req := requestinfo.FromContext(ctx)

	userID := req.UserID
	if userID != nil {
		span.SetAttributes(attribute.String("user.id", *userID))
	}

	var language *string
	if input.Language != nil {
		language = new(string)
		*language = input.Language.String()
	}
	log.Info().Any("userid", userID).Msg("User ID from context")
	log.Info().Any("language", language).Msg("language")
	updatedUser, err := userService.UpdateUser(ctx, *userID, input.Username, input.Firstname, input.Lastname, language, input.Email)

	if err != nil {
		metrics.GetAppMetrics().ResolverMetric(
			float64(time.Since(startTime).Milliseconds()),
			"UpdateUser",
			metrics.Error,
		)
		return nil, err
	}

	var userLanguage model.Language
	if updatedUser.Language != "" {
		userLanguage = model.Language(updatedUser.Language)
	}

	metrics.GetAppMetrics().ResolverMetric(
		float64(time.Since(startTime).Milliseconds()),
		"UpdateUser",
		metrics.Success,
	)

	return &model.User{
		ID:              updatedUser.ID,
		Firstname:       updatedUser.FirstName,
		Lastname:        updatedUser.LastName,
		Username:        updatedUser.Username,
		Language:        userLanguage,
		Email:           updatedUser.Email,
		ProfileImageURL: updatedUser.ProfileImageURL,
	}, nil
}
