package handlers

import (
	"context"
	"fmt"
	"github.com/99designs/gqlgen/graphql"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/apollotracing"
	"github.com/sirupsen/logrus"

	"github.com/weeb-vip/user/config"
	"github.com/weeb-vip/user/graph"
	"github.com/weeb-vip/user/graph/generated"
	"github.com/weeb-vip/user/http/handlers/logger"
	"github.com/weeb-vip/user/http/handlers/metrics"
	"github.com/weeb-vip/user/http/handlers/requestinfo"
	"github.com/weeb-vip/user/internal/jwt"
	"github.com/weeb-vip/user/internal/measurements"
	"github.com/weeb-vip/user/internal/services/users"
)

func BuildRootHandler(tokenizer jwt.Tokenizer) http.Handler { // nolint
	logrus.SetFormatter(&logrus.TextFormatter{})

	conf, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	userService := users.NewUserService()
	resolvers := &graph.Resolver{
		UserService:  userService,
		JwtTokenizer: tokenizer,
		Config:       *conf,
	}
	cfg := generated.Config{Resolvers: resolvers}
	cfg.Directives.Authenticated = func(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error) {
		req := requestinfo.FromContext(ctx)

		if req.UserID == nil {
			// unauthorized
			return nil, fmt.Errorf("Access denied")
		}

		return next(ctx)
	}
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(cfg))
	srv.Use(apollotracing.Tracer{})

	client := measurements.New()

	return requestinfo.Handler()(logger.Handler()(metrics.Handler(client)(srv)))
}
