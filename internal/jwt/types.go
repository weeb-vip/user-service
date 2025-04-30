package jwt

import (
	"time"

	"github.com/weeb-vip/user/internal/keypair"
)

type tokenizer struct {
	signingKey keypair.RotatingSigningKey
}

type Claims struct {
	Subject      *string
	TTL          *time.Duration
	Purpose      *string
	RefreshToken *string
}

type Tokenizer interface {
	Tokenize(claims Claims) (string, error)
}
