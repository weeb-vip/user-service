package follows

import (
	"context"

	"gorm.io/gorm"

	followmodels "github.com/weeb-vip/user-service/internal/services/follows/models"
	usermodels "github.com/weeb-vip/user-service/internal/services/users/models"
)

// ViewerStatus is how the current viewer relates to another user. It mirrors
// the GraphQL FollowStatus enum one to one.
type ViewerStatus string

const (
	ViewerNone      ViewerStatus = "NONE"
	ViewerRequested ViewerStatus = "REQUESTED"
	ViewerFollowing ViewerStatus = "FOLLOWING"
	ViewerSelf      ViewerStatus = "SELF"
)

// Page is one page of users with the total count.
type Page struct {
	Users []*usermodels.User
	Total int64
}

// Follows is the follow graph's behaviour: who may follow whom, when approval
// is needed, who may see a list, and which event each change announces.
type Follows interface {
	// Follow creates the edge viewer -> target and reports whether it is live
	// (ViewerFollowing) or waiting for approval (ViewerRequested). Repeating
	// it is a no-op that reports the current state.
	Follow(ctx context.Context, viewerID, targetID string) (ViewerStatus, error)
	// Unfollow removes the edge viewer -> target, pending or accepted, and
	// reports whether there was one.
	Unfollow(ctx context.Context, viewerID, targetID string) (bool, error)
	// Accept approves the pending request follower -> owner.
	Accept(ctx context.Context, ownerID, followerID string) error
	// Decline rejects the pending request follower -> owner.
	Decline(ctx context.Context, ownerID, followerID string) error
	// RemoveFollower breaks the accepted edge follower -> owner.
	RemoveFollower(ctx context.Context, ownerID, followerID string) error

	// Followers and Following list accepted edges of userID, subject to the
	// visibility rule: a user who requires approval shows their lists only to
	// themselves and to accepted followers. Others get an empty page with the
	// true total, so counts stay public while names do not.
	Followers(ctx context.Context, viewerID *string, userID string, page, limit int) (Page, error)
	Following(ctx context.Context, viewerID *string, userID string, page, limit int) (Page, error)
	// Requests lists pending requests addressed to ownerID.
	Requests(ctx context.Context, ownerID string, page, limit int) (Page, error)

	Counts(ctx context.Context, userID string) (followmodels.Counts, error)
	ViewerStatus(ctx context.Context, viewerID *string, targetID string) (ViewerStatus, error)
	// FollowerIDs pages accepted follower ids for fan-out. See the repository.
	FollowerIDs(ctx context.Context, userID string, after string, limit int) ([]string, error)
}

// EventWriter records an event in the caller's transaction. It is the one
// seam between this service and go-outbox-lib, so tests can assert on what
// would be published without a database.
type EventWriter interface {
	Write(ctx context.Context, tx *gorm.DB, subject string, id string, payload any) error
}

// UserLookup is the slice of the users service the follow graph needs.
type UserLookup interface {
	GetUserDetails(ctx context.Context, id string) (*usermodels.User, error)
}
