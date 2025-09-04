package graph

import (
	"github.com/weeb-vip/user-service/config"
	"github.com/weeb-vip/user-service/internal/jwt"
	"github.com/weeb-vip/user-service/internal/services/image"
	"github.com/weeb-vip/user-service/internal/services/users"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	UserService  users.User
	JwtTokenizer jwt.Tokenizer
	Config       config.Config
	ImageService *image.ImageService
}
