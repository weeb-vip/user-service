package resolvers

import (
	"context"
	"errors"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/weeb-vip/user-service/internal/entities"
	"github.com/weeb-vip/user-service/internal/services/users"
)

// ErrorPresenter is the single place every GraphQL error acquires the API's
// standard envelope: a user-safe `message`, a stable machine `code` (never an
// HTTP status), and the raw `error` string for debugging.
//
// Wiring it once on the server is what lets the meaning live at the lowest
// level: a typed error returned from the service or repository bubbles up
// through its resolver untouched and comes out here in the right shape, so no
// resolver has to wrap anything itself.
func ErrorPresenter(ctx context.Context, e error) *gqlerror.Error {
	err := graphql.DefaultErrorPresenter(ctx, e)

	// A users-service error carries the meaning: its own code and a message
	// already written for a person.
	var svc *users.Error
	if errors.As(e, &svc) {
		return envelope(err, svc.Message, svc.Code.String(), e.Error())
	}

	// An entity-layer error likewise names its own code.
	var ent *entities.ServiceError
	if errors.As(e, &ent) {
		return envelope(err, ent.Message, ent.Code, e.Error())
	}

	// Already shaped with a code (e.g. an auth-directive rejection): keep the
	// code, fill in only the envelope fields that are missing.
	if code, ok := extString(err.Extensions, "code"); ok && code != "" {
		msg := err.Message
		if m, ok := extString(err.Extensions, "message"); ok && m != "" {
			msg = m
		}
		return envelope(err, msg, code, e.Error())
	}

	// Unclassified: a safe generic message, with the raw detail kept in `error`
	// rather than surfaced as the message a user reads.
	return envelope(err, "Something went wrong", "UNKNOWN_ERROR", e.Error())
}

func envelope(err *gqlerror.Error, message, code, raw string) *gqlerror.Error {
	err.Message = message
	if err.Extensions == nil {
		err.Extensions = map[string]interface{}{}
	}
	err.Extensions["message"] = message
	err.Extensions["code"] = code
	err.Extensions["error"] = raw
	return err
}

func extString(m map[string]interface{}, k string) (string, bool) {
	if m == nil {
		return "", false
	}
	v, ok := m[k].(string)
	return v, ok
}
