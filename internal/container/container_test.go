package container_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/weeb-vip/user/internal/container"
	"github.com/weeb-vip/user/internal/keypair"
)

func TestNew(t *testing.T) {
	t.Run("container can replace item and get latest item at any point and return correct data", func(t *testing.T) {
		c := container.New(keypair.SigningKey{
			Key: "key-1",
			ID:  "1",
		})
		assert.Equal(t, "key-1", c.GetLatest().Key)
		assert.Equal(t, "1", c.GetLatest().ID)
		c.ReplaceWith(keypair.SigningKey{
			Key: "key-2",
			ID:  "2",
		})
		assert.Equal(t, "key-2", c.GetLatest().Key)
		assert.Equal(t, "2", c.GetLatest().ID)
	})
}
