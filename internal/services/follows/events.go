package follows

import "time"

// Subject is the NATS subject every follow event is published on.
const Subject = "user-follow"

// Event types. A consumer switches on these; the names say what happened, not
// what to do about it, so a new consumer can decide for itself.
const (
	// EventFollowRequested: follower asked to follow a user who requires approval.
	EventFollowRequested = "follow_requested"
	// EventFollowAccepted: the edge is live. AutoAccepted tells whether the
	// followee approved it (false) or never had to (true).
	EventFollowAccepted = "follow_accepted"
	// EventFollowDeclined: the followee turned a pending request down.
	EventFollowDeclined = "follow_declined"
	// EventUnfollowed: the follower withdrew, whether the edge was pending or live.
	EventUnfollowed = "unfollowed"
	// EventFollowerRemoved: the followee removed an accepted follower.
	EventFollowerRemoved = "follower_removed"
)

// Event is the payload on Subject.
//
// ID is also the outbox row id and the JetStream message id, so a consumer
// can be idempotent on the body alone. PreviousStatus is set on removals so a
// consumer can tell a withdrawn request from a broken follow.
type Event struct {
	ID             string    `json:"id"`
	Type           string    `json:"type"`
	FollowerID     string    `json:"follower_id"`
	FolloweeID     string    `json:"followee_id"`
	AutoAccepted   bool      `json:"auto_accepted,omitempty"`
	PreviousStatus string    `json:"previous_status,omitempty"`
	OccurredAt     time.Time `json:"occurred_at"`
}
