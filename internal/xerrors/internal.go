package xerrors

import (
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func gqlError(message, code string) *gqlerror.Error {
	return &gqlerror.Error{
		Message:    message,
		Extensions: map[string]interface{}{"code": code},
	}
}

func ServiceError(message string, code string) *gqlerror.Error {
	return &gqlerror.Error{
		Message: message,
		Extensions: map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
}

// CustomError is the standard failure shape the API returns: a human-readable
// message, a stable machine code (not an HTTP status), and the underlying error
// string for debugging. Clients switch on `code`; `message` is safe to show a
// user; `error` is the raw detail.
func CustomError(message, code, errStr string) *gqlerror.Error {
	return &gqlerror.Error{
		Message: message,
		Extensions: map[string]interface{}{
			"message": message,
			"code":    code,
			"error":   errStr,
		},
	}
}

func ChallengeError(
	message string,
	code string,
	flow string,
	challenge string,
	requestType string,
	metadata *interface{},
) *gqlerror.Error {
	return &gqlerror.Error{
		Message: message,
		Extensions: map[string]interface{}{
			"code":        code,
			"flow":        flow,
			"challenge":   challenge,
			"requestType": requestType,
			"metadata":    metadata,
		},
	}
}
func NotFound(message string) *gqlerror.Error {
	return gqlError(message, "not-found")
}

func DBError(message string) *gqlerror.Error {
	return gqlError(message, "memstore-error")
}

func DBFetchError() *gqlerror.Error {
	return DBError("Fetching from DB failed")
}

func InternalError(message string) *gqlerror.Error {
	return gqlError(message, "internal-error")
}

func Forbidden(message string) *gqlerror.Error {
	return gqlError(message, "forbidden")
}
