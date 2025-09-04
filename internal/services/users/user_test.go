package users_test

import (
	"context"
	"github.com/weeb-vip/user-service/internal/services/users"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserService_AddUser(t *testing.T) {
	t.Parallel()
	t.Run("Test AddUser", func(t *testing.T) {
		t.Parallel()
		a := assert.New(t)

		credentialService := users.NewUserService()

		_, err := credentialService.AddUser(context.TODO(), "1", "username", "first", "last", "en")
		a.NoError(err)
	})
	t.Run("Test AddUser 2 times - idempotence", func(t *testing.T) {
		t.Parallel()
		a := assert.New(t)

		credentialService := users.NewUserService()

		_, err := credentialService.AddUser(context.TODO(), "1", "username", "first", "last", "en")
		a.NoError(err)
		_, err = credentialService.AddUser(context.TODO(), "2", "username2", "first", "last", "en")
		a.NoError(err)
	})
	t.Run("Test AddUser 2 times with same username", func(t *testing.T) {
		t.Parallel()
		a := assert.New(t)

		credentialService := users.NewUserService()

		_, err := credentialService.AddUser(context.TODO(), "1", "username", "first", "last", "en")
		a.NoError(err)
		_, err = credentialService.AddUser(context.TODO(), "1", "username", "first", "last", "en")
		a.Error(err)
	})
}

func TestUserService_GetUserDetails(t *testing.T) {
	t.Parallel()
	t.Run("Test GetDetails", func(t *testing.T) {
		t.Parallel()
		a := assert.New(t)

		credentialService := users.NewUserService()

		_, err := credentialService.AddUser(context.TODO(), "1", "username", "first", "last", "en")
		a.NoError(err)

		a.NotNil(credentialService.GetUserDetails(context.TODO(), "username"))
		a.Nil(credentialService.GetUserDetails(context.TODO(), "username2"))
	})
}
