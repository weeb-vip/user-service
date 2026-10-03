package follows

import "github.com/weeb-vip/user-service/internal/entities"

// Error codes surfaced to clients through the ErrorPresenter's envelope.
const (
	CodeSelfFollow      = "FOLLOW_SELF"
	CodeUserNotFound    = "USER_NOT_FOUND"
	CodeRequestNotFound = "FOLLOW_REQUEST_NOT_FOUND"
	CodeFollowNotFound  = "FOLLOW_NOT_FOUND"
	CodeForbidden       = "FOLLOW_FORBIDDEN"
)

func errSelfFollow() error {
	return &entities.ServiceError{Code: CodeSelfFollow, Message: "You cannot follow yourself"}
}

func errUserNotFound() error {
	return &entities.ServiceError{Code: CodeUserNotFound, Message: "That user does not exist"}
}

func errRequestNotFound() error {
	return &entities.ServiceError{Code: CodeRequestNotFound, Message: "There is no pending follow request from that user"}
}

func errFollowNotFound() error {
	return &entities.ServiceError{Code: CodeFollowNotFound, Message: "That user does not follow you"}
}
