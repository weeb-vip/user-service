package users

const (
	UserErrorInternalError ErrorCode = "INTERNAL_ERROR"      // nolint
	UserErrorUserExists    ErrorCode = "USER_EXISTS"         // nolint
	UserErrorInvalidUsers  ErrorCode = "INVALID_CREDENTIALS" // nolint
)

type ErrorCode string

type Error struct {
	Code    ErrorCode
	Message string
}

func (c ErrorCode) String() string {
	return string(c)
}

func (e Error) Error() string {
	return e.Message
}
