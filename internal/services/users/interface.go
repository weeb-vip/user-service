package users

import (
	"context"

	"github.com/weeb-vip/user/internal/services/users/models"
)

type User interface {
	AddUser(ctx context.Context, id string, username string, firstName string, lastName string, language string) (*models.User, error)
	GetUserDetails(ctx context.Context, id string) (*models.User, error)
	UpdateUser(ctx context.Context, id string, username *string, firstName *string, lastName *string, language *string, email *string) (*models.User, error)
}
