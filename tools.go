//go:build tools
// +build tools

package user

import (
	_ "github.com/99designs/gqlgen"
	_ "github.com/golang-migrate/migrate/v4/cmd/migrate"
)
