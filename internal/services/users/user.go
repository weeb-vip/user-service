package users

import (
	"context"
	"strings"
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

	// Identity here is the user ID from the auth service, not the username.
	//
	// user-created events carry no username, so every one of them arrives with
	// an empty string. Matching existence on username meant the first row that
	// happened to have an empty username matched every subsequent event, and
	// no user could be created after it: the events failed, were pushed onto
	// the retry queue, and stayed there because nothing consumes it. The ID is
	// the primary key and is always present, so it is the only safe key here.
	user, err := service.usersRepository.GetUserById(ctx, id)

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

	// Check the ID rather than the pointer. GetUserById does not share
	// GetUserByUsername's convention of returning nil when nothing matched: it
	// returns a pointer to a zero-valued User even on ErrRecordNotFound, so a
	// nil check here reports that every user already exists. Other callers rely
	// on that behaviour, so this reads the ID instead of changing it for them.
	if user != nil && user.ID != "" {
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

	// A username is optional when the account is first created from an event,
	// but it still has to stay unique once somebody sets one. Only check when
	// there is something to check -- an empty username is the normal state for
	// a freshly created account, not a collision.
	if username != "" {
		existing, err := service.usersRepository.GetUserByUsername(ctx, username)
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
		if existing != nil {
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

	// A username identifies a public page, so it has to stay unique. Check
	// before writing to give a clean "taken" error rather than a raw database
	// failure; the unique index below is still the real guarantee, since two
	// requests can pass this check at the same moment.
	if username != nil && *username != "" {
		existing, lookupErr := service.usersRepository.GetUserByUsername(ctx, *username)
		if lookupErr != nil {
			metrics.GetAppMetrics().ServiceMetric(
				float64(time.Since(startTime).Milliseconds()),
				"users",
				"UpdateUser",
				metrics.Error,
			)
			return nil, &Error{Code: UserErrorInternalError, Message: "database error"}
		}
		if existing != nil && existing.ID != id {
			metrics.GetAppMetrics().ServiceMetric(
				float64(time.Since(startTime).Milliseconds()),
				"users",
				"UpdateUser",
				metrics.Error,
			)
			return nil, &Error{Code: UserErrorUsernameTaken, Message: "That username is already taken"}
		}
	}

	result, err := service.usersRepository.UpdateUser(ctx, id, username, firstName, lastName, language, email)

	// A unique-index violation means someone took the name in the gap between
	// the check above and this write -- surface it as the same friendly error,
	// not an opaque internal one. 23505 is Postgres's unique_violation SQLSTATE.
	if err != nil && strings.Contains(err.Error(), "23505") {
		metrics.GetAppMetrics().ServiceMetric(
			float64(time.Since(startTime).Milliseconds()),
			"users",
			"UpdateUser",
			metrics.Error,
		)
		return nil, &Error{Code: UserErrorUsernameTaken, Message: "That username is already taken"}
	}

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

// UpdateBannerImageURL persists the banner path via the repository.
func (service *usersService) UpdateBannerImageURL(ctx context.Context, id string, bannerImageURL string) (*models.User, error) {
	tracer := tracing.GetTracer(ctx)
	ctx, span := tracer.Start(ctx, "service.UpdateBannerImageURL",
		trace.WithAttributes(
			attribute.String("user.id", id),
			attribute.String("service", "users"),
			attribute.String("method", "UpdateBannerImageURL"),
		),
		tracing.GetEnvironmentAttribute(),
	)
	defer span.End()

	return service.usersRepository.UpdateBannerImageURL(ctx, id, bannerImageURL)
}

// UpdateCustomization persists the page's bio, accent and list visibility.
func (service *usersService) UpdateCustomization(ctx context.Context, id string, bio *string, accentColor *string, listsPublic *bool, followApprovalRequired *bool) (*models.User, error) {
	tracer := tracing.GetTracer(ctx)
	ctx, span := tracer.Start(ctx, "service.UpdateCustomization",
		trace.WithAttributes(
			attribute.String("user.id", id),
			attribute.String("service", "users"),
			attribute.String("method", "UpdateCustomization"),
		),
		tracing.GetEnvironmentAttribute(),
	)
	defer span.End()

	return service.usersRepository.UpdateCustomization(ctx, id, bio, accentColor, listsPublic, followApprovalRequired)
}

// GetUserByUsername backs the public page lookup. Nil, nil when nothing matches,
// which the resolver turns into a 404-shaped null rather than an error.
func (service *usersService) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	tracer := tracing.GetTracer(ctx)
	ctx, span := tracer.Start(ctx, "service.GetUserByUsername",
		trace.WithAttributes(
			attribute.String("service", "users"),
			attribute.String("method", "GetUserByUsername"),
		),
		tracing.GetEnvironmentAttribute(),
	)
	defer span.End()

	return service.usersRepository.GetUserByUsername(ctx, username)
}
