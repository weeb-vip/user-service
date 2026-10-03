package follows

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	followmodels "github.com/weeb-vip/user-service/internal/services/follows/models"
	"github.com/weeb-vip/user-service/internal/services/follows/repositories"
	usermodels "github.com/weeb-vip/user-service/internal/services/users/models"
	"github.com/weeb-vip/user-service/metrics"
	"github.com/weeb-vip/user-service/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type service struct {
	repo   repositories.FollowsRepository
	users  UserLookup
	events EventWriter
	now    func() time.Time
	newID  func() string
}

// New wires the service with its dependencies.
func New(repo repositories.FollowsRepository, users UserLookup, events EventWriter) Follows {
	return &service{
		repo:   repo,
		users:  users,
		events: events,
		now:    func() time.Time { return time.Now().UTC() },
		newID:  func() string { return uuid.New().String() },
	}
}

func (s *service) span(ctx context.Context, method string, attrs ...attribute.KeyValue) (context.Context, trace.Span, func(error)) {
	attrs = append(attrs, attribute.String("service", "follows"), attribute.String("method", method))
	ctx, span := tracing.GetTracer(ctx).Start(ctx, "service.follows."+method,
		trace.WithAttributes(attrs...), tracing.GetEnvironmentAttribute())
	start := time.Now()

	return ctx, span, func(err error) {
		result := metrics.Success
		if err != nil {
			result = metrics.Error
			span.RecordError(err)
		}
		metrics.GetAppMetrics().ServiceMetric(float64(time.Since(start).Microseconds())/1000, "follows", method, result)
		span.End()
	}
}

// lookup returns the user or a not-found error. GetUserDetails returns a
// zero-valued user rather than nil when nothing matched (see users.AddUser),
// so the ID is what says whether the user exists.
func (s *service) lookup(ctx context.Context, id string) (*usermodels.User, error) {
	user, err := s.users.GetUserDetails(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil || user.ID == "" {
		return nil, errUserNotFound()
	}

	return user, nil
}

func (s *service) emit(ctx context.Context, tx *gorm.DB, event Event) error {
	event.ID = s.newID()
	event.OccurredAt = s.now()

	return s.events.Write(ctx, tx, Subject, event.ID, event)
}

func (s *service) Follow(ctx context.Context, viewerID, targetID string) (ViewerStatus, error) {
	ctx, _, done := s.span(ctx, "Follow", attribute.String("user.id", viewerID), attribute.String("target.id", targetID))
	var err error
	defer func() { done(err) }()

	if viewerID == targetID {
		err = errSelfFollow()

		return "", err
	}

	target, err := s.lookup(ctx, targetID)
	if err != nil {
		return "", err
	}

	existing, err := s.repo.Get(ctx, viewerID, targetID)
	if err != nil {
		return "", err
	}
	if existing != nil {
		if existing.Status == followmodels.StatusAccepted {
			return ViewerFollowing, nil
		}

		return ViewerRequested, nil
	}

	status := followmodels.StatusAccepted
	eventType := EventFollowAccepted
	result := ViewerFollowing
	if target.FollowApprovalRequired {
		status = followmodels.StatusPending
		eventType = EventFollowRequested
		result = ViewerRequested
	}

	now := s.now()
	follow := &followmodels.UserFollow{
		FollowerID: viewerID,
		FolloweeID: targetID,
		Status:     status,
		CreatedAt:  now,
	}
	if status == followmodels.StatusAccepted {
		follow.AcceptedAt = &now
	}

	err = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.repo.Create(ctx, tx, follow); err != nil {
			return err
		}

		return s.emit(ctx, tx, Event{
			Type:         eventType,
			FollowerID:   viewerID,
			FolloweeID:   targetID,
			AutoAccepted: status == followmodels.StatusAccepted,
		})
	})
	if err != nil {
		return "", err
	}

	return result, nil
}

func (s *service) Unfollow(ctx context.Context, viewerID, targetID string) (bool, error) {
	ctx, _, done := s.span(ctx, "Unfollow", attribute.String("user.id", viewerID), attribute.String("target.id", targetID))
	var err error
	defer func() { done(err) }()

	existing, err := s.repo.Get(ctx, viewerID, targetID)
	if err != nil {
		return false, err
	}
	if existing == nil {
		return false, nil
	}

	removed := false
	err = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		deleted, err := s.repo.Delete(ctx, tx, viewerID, targetID, "")
		if err != nil {
			return err
		}
		if deleted == 0 {
			return nil
		}
		removed = true

		return s.emit(ctx, tx, Event{
			Type:           EventUnfollowed,
			FollowerID:     viewerID,
			FolloweeID:     targetID,
			PreviousStatus: string(existing.Status),
		})
	})

	return removed, err
}

func (s *service) Accept(ctx context.Context, ownerID, followerID string) error {
	ctx, _, done := s.span(ctx, "Accept", attribute.String("user.id", ownerID), attribute.String("follower.id", followerID))
	var err error
	defer func() { done(err) }()

	err = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		updated, err := s.repo.Accept(ctx, tx, followerID, ownerID, s.now())
		if err != nil {
			return err
		}
		if updated == 0 {
			return errRequestNotFound()
		}

		return s.emit(ctx, tx, Event{
			Type:       EventFollowAccepted,
			FollowerID: followerID,
			FolloweeID: ownerID,
		})
	})

	return err
}

func (s *service) Decline(ctx context.Context, ownerID, followerID string) error {
	ctx, _, done := s.span(ctx, "Decline", attribute.String("user.id", ownerID), attribute.String("follower.id", followerID))
	var err error
	defer func() { done(err) }()

	err = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		deleted, err := s.repo.Delete(ctx, tx, followerID, ownerID, followmodels.StatusPending)
		if err != nil {
			return err
		}
		if deleted == 0 {
			return errRequestNotFound()
		}

		return s.emit(ctx, tx, Event{
			Type:           EventFollowDeclined,
			FollowerID:     followerID,
			FolloweeID:     ownerID,
			PreviousStatus: string(followmodels.StatusPending),
		})
	})

	return err
}

func (s *service) RemoveFollower(ctx context.Context, ownerID, followerID string) error {
	ctx, _, done := s.span(ctx, "RemoveFollower", attribute.String("user.id", ownerID), attribute.String("follower.id", followerID))
	var err error
	defer func() { done(err) }()

	err = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		deleted, err := s.repo.Delete(ctx, tx, followerID, ownerID, followmodels.StatusAccepted)
		if err != nil {
			return err
		}
		if deleted == 0 {
			return errFollowNotFound()
		}

		return s.emit(ctx, tx, Event{
			Type:           EventFollowerRemoved,
			FollowerID:     followerID,
			FolloweeID:     ownerID,
			PreviousStatus: string(followmodels.StatusAccepted),
		})
	})

	return err
}

// canSeeLists applies the visibility rule for a user's follower and following
// lists. Everyone sees them unless the user requires approval to be followed,
// in which case only the user and their accepted followers do.
func (s *service) canSeeLists(ctx context.Context, viewerID *string, user *usermodels.User) (bool, error) {
	if !user.FollowApprovalRequired {
		return true, nil
	}
	if viewerID == nil {
		return false, nil
	}
	if *viewerID == user.ID {
		return true, nil
	}

	edge, err := s.repo.Get(ctx, *viewerID, user.ID)
	if err != nil {
		return false, err
	}

	return edge != nil && edge.Status == followmodels.StatusAccepted, nil
}

func (s *service) list(ctx context.Context, viewerID *string, userID string, page, limit int,
	fetch func(ctx context.Context) ([]*usermodels.User, int64, error),
	total func(ctx context.Context) (int64, error),
) (Page, error) {
	user, err := s.lookup(ctx, userID)
	if err != nil {
		return Page{}, err
	}

	visible, err := s.canSeeLists(ctx, viewerID, user)
	if err != nil {
		return Page{}, err
	}
	if !visible {
		n, err := total(ctx)
		if err != nil {
			return Page{}, err
		}

		return Page{Users: []*usermodels.User{}, Total: n}, nil
	}

	users, n, err := fetch(ctx)
	if err != nil {
		return Page{}, err
	}
	if users == nil {
		users = []*usermodels.User{}
	}

	return Page{Users: users, Total: n}, nil
}

func (s *service) Followers(ctx context.Context, viewerID *string, userID string, page, limit int) (Page, error) {
	ctx, _, done := s.span(ctx, "Followers", attribute.String("target.id", userID))
	var err error
	defer func() { done(err) }()

	var result Page
	result, err = s.list(ctx, viewerID, userID, page, limit,
		func(ctx context.Context) ([]*usermodels.User, int64, error) {
			return s.repo.ListFollowers(ctx, userID, followmodels.StatusAccepted, page, limit)
		},
		func(ctx context.Context) (int64, error) {
			counts, err := s.repo.Counts(ctx, userID)

			return counts.Followers, err
		},
	)

	return result, err
}

func (s *service) Following(ctx context.Context, viewerID *string, userID string, page, limit int) (Page, error) {
	ctx, _, done := s.span(ctx, "Following", attribute.String("target.id", userID))
	var err error
	defer func() { done(err) }()

	var result Page
	result, err = s.list(ctx, viewerID, userID, page, limit,
		func(ctx context.Context) ([]*usermodels.User, int64, error) {
			return s.repo.ListFollowing(ctx, userID, followmodels.StatusAccepted, page, limit)
		},
		func(ctx context.Context) (int64, error) {
			counts, err := s.repo.Counts(ctx, userID)

			return counts.Following, err
		},
	)

	return result, err
}

func (s *service) Requests(ctx context.Context, ownerID string, page, limit int) (Page, error) {
	ctx, _, done := s.span(ctx, "Requests", attribute.String("user.id", ownerID))
	var err error
	defer func() { done(err) }()

	users, total, err := s.repo.ListFollowers(ctx, ownerID, followmodels.StatusPending, page, limit)
	if err != nil {
		return Page{}, err
	}
	if users == nil {
		users = []*usermodels.User{}
	}

	return Page{Users: users, Total: total}, nil
}

func (s *service) Counts(ctx context.Context, userID string) (followmodels.Counts, error) {
	ctx, _, done := s.span(ctx, "Counts", attribute.String("target.id", userID))
	var err error
	defer func() { done(err) }()

	var counts followmodels.Counts
	counts, err = s.repo.Counts(ctx, userID)

	return counts, err
}

func (s *service) ViewerStatus(ctx context.Context, viewerID *string, targetID string) (ViewerStatus, error) {
	ctx, _, done := s.span(ctx, "ViewerStatus", attribute.String("target.id", targetID))
	var err error
	defer func() { done(err) }()

	if viewerID == nil {
		return ViewerNone, nil
	}
	if *viewerID == targetID {
		return ViewerSelf, nil
	}

	edge, err := s.repo.Get(ctx, *viewerID, targetID)
	if err != nil {
		return "", err
	}
	switch {
	case edge == nil:
		return ViewerNone, nil
	case edge.Status == followmodels.StatusAccepted:
		return ViewerFollowing, nil
	default:
		return ViewerRequested, nil
	}
}

func (s *service) FollowerIDs(ctx context.Context, userID string, after string, limit int) ([]string, error) {
	ctx, _, done := s.span(ctx, "FollowerIDs", attribute.String("target.id", userID))
	var err error
	defer func() { done(err) }()

	var ids []string
	ids, err = s.repo.FollowerIDs(ctx, userID, after, limit)
	if ids == nil {
		ids = []string{}
	}

	return ids, err
}
