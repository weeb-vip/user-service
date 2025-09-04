package ulid_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/weeb-vip/user-service/internal/ulid"
)

func TestNew(t *testing.T) {
	id := ulid.New("entity")
	assert.True(t, strings.HasPrefix(id, "entity_"))
	assert.NotEqual(t, id, ulid.New("entity"))
}
