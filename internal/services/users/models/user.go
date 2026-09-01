package models

import (
	"github.com/weeb-vip/user-service/internal/db"
)

type User struct {
	db.BaseModel
	// Nullable: an account created from a user-created event has no username
	// until the person picks one, and that unset state is NULL -- not "" -- so a
	// unique index can allow any number of not-yet-named accounts while still
	// rejecting two people who choose the same name.
	Username        *string `json:"username" gorm:"column:username"`
	FirstName       string  `json:"first_name"`
	LastName        string  `json:"last_name"`
	Language        string  `json:"language"`
	Email           *string `json:"email"`
	ProfileImageURL *string `json:"profile_image_url" gorm:"column:profile_image_url"`
	BannerImageURL  *string `json:"banner_image_url" gorm:"column:banner_image_url"`
	Bio             *string `json:"bio" gorm:"column:bio"`
	AccentColor     *string `json:"accent_color" gorm:"column:accent_color"`
	ListsPublic     bool    `json:"lists_public" gorm:"column:lists_public;not null;default:false"`
}
