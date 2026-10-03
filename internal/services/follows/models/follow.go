package models

import "time"

// Status of a follow edge.
//
// Stored as text rather than a Postgres enum so adding a state is a CHECK
// constraint change, not a type migration.
type Status string

const (
	// StatusPending is a request the followee has not yet approved.
	StatusPending Status = "pending"
	// StatusAccepted is a live follow: the follower sees the followee's activity.
	StatusAccepted Status = "accepted"
)

// UserFollow is one directed edge in the follow graph.
type UserFollow struct {
	FollowerID string     `gorm:"column:follower_id;primaryKey"`
	FolloweeID string     `gorm:"column:followee_id;primaryKey"`
	Status     Status     `gorm:"column:status;not null"`
	CreatedAt  time.Time  `gorm:"column:created_at;not null"`
	AcceptedAt *time.Time `gorm:"column:accepted_at"`
}

// TableName implements gorm's Tabler.
func (UserFollow) TableName() string { return "user_follows" }

// Counts is how many accepted edges touch a user in each direction.
type Counts struct {
	Followers int64
	Following int64
}
