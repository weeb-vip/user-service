package keypair_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/weeb-vip/user-service/internal/keypair"
)

func TestGenerateKeyPair(t *testing.T) {
	t.Run("can generate a valid key pair", func(t *testing.T) {
		key, err := keypair.GenerateKeyPair()
		assert.NoError(t, err)
		assert.NotNil(t, key)
	})
}
