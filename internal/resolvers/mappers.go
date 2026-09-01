package resolvers

import (
	"github.com/weeb-vip/user-service/graph/model"
	"github.com/weeb-vip/user-service/internal/services/users/models"
)

// toGraphUser is the one place the stored user becomes the API user, so a field
// added to the profile is exposed everywhere it is returned rather than in
// whichever resolver happened to be edited.
func toGraphUser(u *models.User) *model.User {
	return &model.User{
		ID:              u.ID,
		Firstname:       u.FirstName,
		Lastname:        u.LastName,
		Username:        u.Username,
		Language:        model.Language(u.Language),
		Email:           u.Email,
		ProfileImageURL: u.ProfileImageURL,
		BannerImageURL:  u.BannerImageURL,
		Bio:             u.Bio,
		AccentColor:     u.AccentColor,
		ListsPublic:     u.ListsPublic,
	}
}

// toPublicUser is the safe subset anyone may see: no email, no language, no
// sessions. Keeping it a separate mapper is what makes the public query
// unable to leak those even if a future edit adds them to the User type.
func toPublicUser(u *models.User) *model.PublicUser {
	return &model.PublicUser{
		ID:              u.ID,
		Username:        u.Username,
		Firstname:       u.FirstName,
		Lastname:        u.LastName,
		ProfileImageURL: u.ProfileImageURL,
		BannerImageURL:  u.BannerImageURL,
		Bio:             u.Bio,
		AccentColor:     u.AccentColor,
		ListsPublic:     u.ListsPublic,
	}
}
