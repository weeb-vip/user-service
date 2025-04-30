package keypair_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/weeb-vip/user/internal/keypair"
)

func getIDGenerator(id string, err error) keypair.PublicKeyIDGenerator {
	count := 0
	return func(publicKey string) (string, error) {
		count = count + 1
		return fmt.Sprintf(id, count), err
	}
}

func TestNewSigningKeyRotator(t *testing.T) {
	t.Run("if getting ID during startup fails, it returns error", func(t *testing.T) {
		rotatingKeyPair, err := keypair.NewSigningKeyRotator(getIDGenerator("", errors.New("some error")))
		assert.Nil(t, rotatingKeyPair)
		assert.Error(t, err)
	})
	t.Run("starts by creating a new keypair and saving", func(t *testing.T) {
		rotatingKeyPair, err := keypair.NewSigningKeyRotator(getIDGenerator("key_%d", nil))
		assert.NoError(t, err)
		assert.Equal(t, "key_1", rotatingKeyPair.GetLatest().ID)
	})
	t.Run("when called rotate, it rolls the key and we get a new ID", func(t *testing.T) {
		rotatingKeyPair, err := keypair.NewSigningKeyRotator(getIDGenerator("key_%d", nil))
		assert.NoError(t, err)
		assert.Equal(t, "key_1", rotatingKeyPair.GetLatest().ID)
		publicKey := rotatingKeyPair.GetLatest().Key
		rotatingKeyPair.Rotate()
		assert.Equal(t, "key_2", rotatingKeyPair.GetLatest().ID)
		assert.NotEqual(t, publicKey, rotatingKeyPair.GetLatest().Key)
	})

	t.Run("if it fails to generate key, it keeps the old keypair", func(t *testing.T) {
		count := 0
		customFailingIDGenerator := keypair.PublicKeyIDGenerator(func(publicKey string) (string, error) {
			count = count + 1
			if count%3 == 0 {
				return "", errors.New("random error")
			}
			return fmt.Sprintf("key_%d", count), nil
		})
		rotatingKeyPair, err := keypair.NewSigningKeyRotator(customFailingIDGenerator)
		assert.NoError(t, err)
		assert.Equal(t, "key_1", rotatingKeyPair.GetLatest().ID)
		rotatingKeyPair.Rotate()
		assert.Equal(t, "key_2", rotatingKeyPair.GetLatest().ID)
		rotatingKeyPair.Rotate()
		assert.Equal(t, "key_2", rotatingKeyPair.GetLatest().ID)
		rotatingKeyPair.Rotate()
		assert.Equal(t, "key_4", rotatingKeyPair.GetLatest().ID)
		rotatingKeyPair.Rotate()
		assert.Equal(t, "key_5", rotatingKeyPair.GetLatest().ID)
		rotatingKeyPair.Rotate()
		assert.Equal(t, "key_5", rotatingKeyPair.GetLatest().ID)
		rotatingKeyPair.Rotate()
		assert.Equal(t, "key_7", rotatingKeyPair.GetLatest().ID)
	})
}
