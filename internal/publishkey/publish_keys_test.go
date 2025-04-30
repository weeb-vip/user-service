//go:build integration

package publishkey_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/weeb-vip/user/internal/publishkey"
)

// this file is excluded for now in CI, I'm running this as a validation step for myself
// later, we'll most likely introduce proper integration test

func TestNewKeyPublisher(t *testing.T) {
	publisher := publishkey.NewKeyPublisher("http://localhost:5001/graphql")
	id, err := publisher.ToKeyManagementService("my-public-key")
	assert.NoError(t, err)
	assert.NotEmpty(t, id)
}
