package models

import (
	"github.com/weeb-vip/user/internal/db"
)

type User struct {
	db.BaseModel
	Username  string  `json:"username"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Language  string  `json:"language"`
	Email     *string `json:"email"`
}
