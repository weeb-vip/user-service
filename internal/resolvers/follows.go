package resolvers

import (
	"context"
	"time"

	"github.com/weeb-vip/user-service/graph/model"
	"github.com/weeb-vip/user-service/http/handlers/requestinfo"
	"github.com/weeb-vip/user-service/internal/services/follows"
	"github.com/weeb-vip/user-service/internal/services/users"
	"github.com/weeb-vip/user-service/internal/xerrors"
	"github.com/weeb-vip/user-service/metrics"
	"github.com/weeb-vip/user-service/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// timed wraps a resolver with the span and metric every resolver here records.
func timed[T any](ctx context.Context, name string, fn func(ctx context.Context) (T, error)) (T, error) {
	ctx, span := tracing.GetTracer(ctx).Start(ctx, name,
		trace.WithAttributes(attribute.String("resolver.name", name)),
		tracing.GetEnvironmentAttribute(),
	)
	defer span.End()
	start := time.Now()

	out, err := fn(ctx)
	result := metrics.Success
	if err != nil {
		result = metrics.Error
		span.RecordError(err)
	}
	metrics.GetAppMetrics().ResolverMetric(float64(time.Since(start).Microseconds())/1000, name, result)

	return out, err
}

// viewer is the signed-in user, or an UNAUTHORIZED error. The @Authenticated
// directive already rejects anonymous calls to the mutations, so this is a
// second line, not the first.
func viewer(ctx context.Context) (string, error) {
	req := requestinfo.FromContext(ctx)
	if req.UserID == nil {
		return "", xerrors.CustomError("You must be signed in to do that", "UNAUTHORIZED", "access denied")
	}

	return *req.UserID, nil
}

func toFollowStatus(status follows.ViewerStatus) model.FollowStatus {
	switch status {
	case follows.ViewerRequested:
		return model.FollowStatusRequested
	case follows.ViewerFollowing:
		return model.FollowStatusFollowing
	case follows.ViewerSelf:
		return model.FollowStatusSelf
	default:
		return model.FollowStatusNone
	}
}

func toPublicUserPage(page follows.Page, pageNo, limit int) *model.PublicUserPaginated {
	out := &model.PublicUserPaginated{
		Page:  pageNo,
		Limit: limit,
		Total: int(page.Total),
		Users: make([]*model.PublicUser, 0, len(page.Users)),
	}
	for _, u := range page.Users {
		out.Users = append(out.Users, toPublicUser(u))
	}

	return out
}

// PublicUserByID is the by-id twin of UserByUsername, and what the federation
// entity resolver for PublicUser uses.
func PublicUserByID(ctx context.Context, userService users.User, id string) (*model.PublicUser, error) {
	return timed(ctx, "PublicUserByID", func(ctx context.Context) (*model.PublicUser, error) {
		user, err := userService.GetUserDetails(ctx, id)
		if err != nil {
			return nil, err
		}
		if user == nil || user.ID == "" {
			return nil, nil
		}

		return toPublicUser(user), nil
	})
}

func Follow(ctx context.Context, svc follows.Follows, targetID string) (model.FollowStatus, error) {
	return timed(ctx, "Follow", func(ctx context.Context) (model.FollowStatus, error) {
		viewerID, err := viewer(ctx)
		if err != nil {
			return "", err
		}
		status, err := svc.Follow(ctx, viewerID, targetID)
		if err != nil {
			return "", err
		}

		return toFollowStatus(status), nil
	})
}

func Unfollow(ctx context.Context, svc follows.Follows, targetID string) (bool, error) {
	return timed(ctx, "Unfollow", func(ctx context.Context) (bool, error) {
		viewerID, err := viewer(ctx)
		if err != nil {
			return false, err
		}

		return svc.Unfollow(ctx, viewerID, targetID)
	})
}

func AcceptFollowRequest(ctx context.Context, svc follows.Follows, followerID string) (bool, error) {
	return timed(ctx, "AcceptFollowRequest", func(ctx context.Context) (bool, error) {
		ownerID, err := viewer(ctx)
		if err != nil {
			return false, err
		}
		if err := svc.Accept(ctx, ownerID, followerID); err != nil {
			return false, err
		}

		return true, nil
	})
}

func DeclineFollowRequest(ctx context.Context, svc follows.Follows, followerID string) (bool, error) {
	return timed(ctx, "DeclineFollowRequest", func(ctx context.Context) (bool, error) {
		ownerID, err := viewer(ctx)
		if err != nil {
			return false, err
		}
		if err := svc.Decline(ctx, ownerID, followerID); err != nil {
			return false, err
		}

		return true, nil
	})
}

func RemoveFollower(ctx context.Context, svc follows.Follows, followerID string) (bool, error) {
	return timed(ctx, "RemoveFollower", func(ctx context.Context) (bool, error) {
		ownerID, err := viewer(ctx)
		if err != nil {
			return false, err
		}
		if err := svc.RemoveFollower(ctx, ownerID, followerID); err != nil {
			return false, err
		}

		return true, nil
	})
}

func Followers(ctx context.Context, svc follows.Follows, userID string, page, limit int) (*model.PublicUserPaginated, error) {
	return timed(ctx, "Followers", func(ctx context.Context) (*model.PublicUserPaginated, error) {
		result, err := svc.Followers(ctx, requestinfo.FromContext(ctx).UserID, userID, page, limit)
		if err != nil {
			return nil, err
		}

		return toPublicUserPage(result, page, limit), nil
	})
}

func Following(ctx context.Context, svc follows.Follows, userID string, page, limit int) (*model.PublicUserPaginated, error) {
	return timed(ctx, "Following", func(ctx context.Context) (*model.PublicUserPaginated, error) {
		result, err := svc.Following(ctx, requestinfo.FromContext(ctx).UserID, userID, page, limit)
		if err != nil {
			return nil, err
		}

		return toPublicUserPage(result, page, limit), nil
	})
}

func FollowRequests(ctx context.Context, svc follows.Follows, page, limit int) (*model.PublicUserPaginated, error) {
	return timed(ctx, "FollowRequests", func(ctx context.Context) (*model.PublicUserPaginated, error) {
		ownerID, err := viewer(ctx)
		if err != nil {
			return nil, err
		}
		result, err := svc.Requests(ctx, ownerID, page, limit)
		if err != nil {
			return nil, err
		}

		return toPublicUserPage(result, page, limit), nil
	})
}

func FollowerIDs(ctx context.Context, svc follows.Follows, userID string, after *string, limit int) ([]string, error) {
	return timed(ctx, "FollowerIDs", func(ctx context.Context) ([]string, error) {
		cursor := ""
		if after != nil {
			cursor = *after
		}
		if limit > 1000 {
			limit = 1000
		}

		return svc.FollowerIDs(ctx, userID, cursor, limit)
	})
}

// PublicUserCounts backs the followerCount and followingCount fields.
func PublicUserCounts(ctx context.Context, svc follows.Follows, userID string) (followers int, following int, err error) {
	counts, err := svc.Counts(ctx, userID)
	if err != nil {
		return 0, 0, err
	}

	return int(counts.Followers), int(counts.Following), nil
}

// PublicUserViewerFollowStatus backs the viewerFollowStatus field.
func PublicUserViewerFollowStatus(ctx context.Context, svc follows.Follows, userID string) (model.FollowStatus, error) {
	status, err := svc.ViewerStatus(ctx, requestinfo.FromContext(ctx).UserID, userID)
	if err != nil {
		return "", err
	}

	return toFollowStatus(status), nil
}
