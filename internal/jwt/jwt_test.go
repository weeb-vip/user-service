package jwt_test

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/weeb-vip/user-service/internal/jwt"
	"github.com/weeb-vip/user-service/internal/keypair"
)

type mockSigningKeyStruct struct {
	key keypair.SigningKey
}

func (m mockSigningKeyStruct) Rotate() {}

func (m mockSigningKeyStruct) RotateInBackground(every time.Duration) { return }

func (m mockSigningKeyStruct) GetLatest() keypair.SigningKey {
	return m.key
}

func TestNew(t *testing.T) {
	keyPair, keyGenerateError := keypair.GenerateKeyPair()
	assert.NoError(t, keyGenerateError)
	t.Run("signed JWT", func(t *testing.T) {
		tokenizer := jwt.New(mockSigningKeyStruct{key: keypair.SigningKey{Key: keyPair.PrivateKey, ID: "key_id"}})
		token, err := tokenizer.Tokenize(jwt.Claims{
			Subject: getPointer("user_1"),
			TTL:     getPointer(time.Second * 15),
			Purpose: nil,
		})
		t.Run("has kid header", func(t *testing.T) {
			header := strings.Split(token, ".")[0]
			decoded, err := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(header)
			assert.NoError(t, err)
			headerMap := map[string]interface{}{}
			err = json.Unmarshal(decoded, &headerMap)
			assert.Equal(t, "key_id", headerMap["kid"].(string))
			assert.NoError(t, err)
		})
		t.Run("has sub field", func(t *testing.T) {
			header := strings.Split(token, ".")[1]
			decoded, err := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(header)
			assert.NoError(t, err)
			headerMap := map[string]interface{}{}
			err = json.Unmarshal(decoded, &headerMap)
			assert.Equal(t, "user_1", headerMap["sub"].(string))
			assert.NoError(t, err)
		})
		t.Run("does not have purpose field", func(t *testing.T) {
			header := strings.Split(token, ".")[1]
			decoded, err := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(header)
			assert.NoError(t, err)
			headerMap := map[string]interface{}{}
			err = json.Unmarshal(decoded, &headerMap)
			assert.Nil(t, headerMap["purpose"])
			assert.NoError(t, err)
		})
		t.Run("has exp field", func(t *testing.T) {
			header := strings.Split(token, ".")[1]
			decoded, err := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(header)
			assert.NoError(t, err)
			headerMap := map[string]interface{}{}
			err = json.Unmarshal(decoded, &headerMap)
			assert.Equal(t, time.Now().Add(15*time.Second).Unix(), int64(headerMap["exp"].(float64)))
			assert.NoError(t, err)
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
	})
}

func getPointer[T any](val T) *T {
	return &val
}
