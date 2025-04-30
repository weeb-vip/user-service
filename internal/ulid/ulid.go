package ulid

import (
	"crypto/rand"
	"fmt"

	"github.com/oklog/ulid/v2"
)

func New(entity string) string {
	return fmt.Sprintf("%s_%s", entity, ulid.MustNew(ulid.Now(), ulid.Monotonic(rand.Reader, 0)).String())
}
