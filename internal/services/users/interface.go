package users

import (
	"context"

	"github.com/weeb-vip/user-service/internal/services/users/models"
)

type User interface {
	AddUser(ctx context.Context, id string, username string, firstName string, lastName string, language string) (*models.User, error)
	GetUserDetails(ctx context.Context, id string) (*models.User, error)
	UpdateUser(ctx context.Context, id string, username *string, firstName *string, lastName *string, language *string, email *string) (*models.User, error)
	UpdateProfileImageURL(ctx context.Context, id string, profileImageURL string) (*models.User, error)
	UpdateBannerImageURL(ctx context.Context, id string, bannerImageURL string) (*models.User, error)
	UpdateCustomization(ctx context.Context, id string, bio *string, accentColor *string, listsPublic *bool, followApprovalRequired *bool) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
}
