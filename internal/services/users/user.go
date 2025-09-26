package users

import (
	"context"
	"time"

	"github.com/weeb-vip/user-service/internal/services/users/models"
	"github.com/weeb-vip/user-service/internal/services/users/repositories"
	"github.com/weeb-vip/user-service/metrics"
	"github.com/weeb-vip/user-service/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type usersService struct {
	usersRepository repositories.UsersRepository
}

func NewUserService() User {
	usersRepository := repositories.GetUsersRepository()

	return &usersService{
		usersRepository: usersRepository,
	}
}

func (service *usersService) AddUser(
	ctx context.Context,
	id string,
	username string,
	firstName string,
	lastName string,
	language string,
) (*models.User, error) {
	tracer := tracing.GetTracer(ctx)
	ctx, span := tracer.Start(ctx, "service.AddUser",
		trace.WithAttributes(
			attribute.String("user.id", id),
			attribute.String("user.username", username),
			attribute.String("service", "users"),
			attribute.String("method", "AddUser"),
		),
		tracing.GetEnvironmentAttribute(),
	)
	defer span.End()

	startTime := time.Now()

	// check if user already exists
	user, err := service.usersRepository.GetUserByUsername(ctx, username)

	if err != nil {
		metrics.GetAppMetrics().ServiceMetric(
			float64(time.Since(startTime).Milliseconds()),
			"users",
			"AddUser",
			metrics.Error,
		)
		return nil, &Error{
			Code:    UserErrorInternalError,
			Message: "database error",
		}
	}

	if user != nil {
		metrics.GetAppMetrics().ServiceMetric(
			float64(time.Since(startTime).Milliseconds()),
			"users",
			"AddUser",
			metrics.Error,
		)
		return nil, &Error{
			Code:    UserErrorUserExists,
			Message: "user already exists",
		}
	}

	result, err := service.usersRepository.AddUser(
		ctx,
		username,
		id,
		firstName,
		lastName,
		language,
	)

	metricResult := metrics.Success
	if err != nil {
		metricResult = metrics.Error
	}
	metrics.GetAppMetrics().ServiceMetric(
		float64(time.Since(startTime).Milliseconds()),
		"users",
		"AddUser",
		metricResult,
	)

	return result, err
}

func (service *usersService) GetUserDetails( //nolint
	ctx context.Context,
	id string,
) (*models.User, error) {
	tracer := tracing.GetTracer(ctx)
	ctx, span := tracer.Start(ctx, "service.GetUserDetails",
		trace.WithAttributes(
			attribute.String("user.id", id),
			attribute.String("service", "users"),
			attribute.String("method", "GetUserDetails"),
		),
		tracing.GetEnvironmentAttribute(),
	)
	defer span.End()

	startTime := time.Now()

	user, err := service.usersRepository.GetUserById(ctx, id) // nolint
	if err != nil {
		metrics.GetAppMetrics().ServiceMetric(
			float64(time.Since(startTime).Milliseconds()),
			"users",
			"GetUserDetails",
			metrics.Error,
		)
		return nil, &Error{
			Code:    UserErrorInternalError,
			Message: "database error",
		}
	}

	if user == nil {
		metrics.GetAppMetrics().ServiceMetric(
			float64(time.Since(startTime).Milliseconds()),
			"users",
			"GetUserDetails",
			metrics.Error,
		)
		return nil, &Error{
			Code:    UserErrorInvalidUsers,
			Message: "invalid user",
		}
	}

	metrics.GetAppMetrics().ServiceMetric(
		float64(time.Since(startTime).Milliseconds()),
		"users",
		"GetUserDetails",
		metrics.Success,
	)

	return user, nil
}

func (service *usersService) UpdateUser(
	ctx context.Context,
	id string,
	username *string,
	firstName *string,
	lastName *string,
	language *string,
	email *string,
) (*models.User, error) {
	tracer := tracing.GetTracer(ctx)
	ctx, span := tracer.Start(ctx, "service.UpdateUser",
		trace.WithAttributes(
			attribute.String("user.id", id),
			attribute.String("service", "users"),
			attribute.String("method", "UpdateUser"),
		),
		tracing.GetEnvironmentAttribute(),
	)
	defer span.End()

	startTime := time.Now()

	result, err := service.usersRepository.UpdateUser(ctx, id, username, firstName, lastName, language, email)

	metricResult := metrics.Success
	if err != nil {
		metricResult = metrics.Error
	}
	metrics.GetAppMetrics().ServiceMetric(
		float64(time.Since(startTime).Milliseconds()),
		"users",
		"UpdateUser",
		metricResult,
	)

	return result, err
}

func (service *usersService) UpdateProfileImageURL(
	ctx context.Context,
	id string,
	profileImageURL string,
) (*models.User, error) {
	tracer := tracing.GetTracer(ctx)
	ctx, span := tracer.Start(ctx, "service.UpdateProfileImageURL",
		trace.WithAttributes(
			attribute.String("user.id", id),
			attribute.String("service", "users"),
			attribute.String("method", "UpdateProfileImageURL"),
			attribute.String("image.url", profileImageURL),
		),
		tracing.GetEnvironmentAttribute(),
	)
	defer span.End()

	startTime := time.Now()

	user, err := service.usersRepository.GetUserById(ctx, id)
	if err != nil {
		metrics.GetAppMetrics().ServiceMetric(
			float64(time.Since(startTime).Milliseconds()),
			"users",
			"UpdateProfileImageURL",
			metrics.Error,
		)
		return nil, &Error{
			Code:    UserErrorInternalError,
			Message: "database error",
		}
	}

	if user == nil {
		metrics.GetAppMetrics().ServiceMetric(
			float64(time.Since(startTime).Milliseconds()),
			"users",
			"UpdateProfileImageURL",
			metrics.Error,
		)
		return nil, &Error{
			Code:    UserErrorInvalidUsers,
			Message: "user not found",
		}
	}

	result, err := service.usersRepository.UpdateProfileImageURL(ctx, id, profileImageURL)

	metricResult := metrics.Success
	if err != nil {
		metricResult = metrics.Error
	}
	metrics.GetAppMetrics().ServiceMetric(
		float64(time.Since(startTime).Milliseconds()),
		"users",
		"UpdateProfileImageURL",
		metricResult,
	)

	return result, err
}
