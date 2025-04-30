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
