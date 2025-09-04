package handlers

import (
	"context"
	"fmt"
	"github.com/99designs/gqlgen/graphql"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/apollotracing"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/sirupsen/logrus"

	"github.com/weeb-vip/user-service/config"
	"github.com/weeb-vip/user-service/graph"
	"github.com/weeb-vip/user-service/graph/generated"
	"github.com/weeb-vip/user-service/http/handlers/logger"
	"github.com/weeb-vip/user-service/http/handlers/metrics"
	"github.com/weeb-vip/user-service/http/handlers/requestinfo"
	"github.com/weeb-vip/user-service/internal/jwt"
	"github.com/weeb-vip/user-service/internal/measurements"
	"github.com/weeb-vip/user-service/internal/services/image"
	"github.com/weeb-vip/user-service/internal/services/users"
	"github.com/weeb-vip/user-service/internal/storage/minio"
)

func BuildRootHandler(tokenizer jwt.Tokenizer) http.Handler { // nolint
	logrus.SetFormatter(&logrus.TextFormatter{})

	conf, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	userService := users.NewUserService()
	
	// Initialize MinIO storage
	minioStorage := minio.NewMinioStorage(conf.MinioConfig)
	imageService := image.NewImageService(minioStorage)
	
	resolvers := &graph.Resolver{
		UserService:  userService,
		JwtTokenizer: tokenizer,
		Config:       *conf,
		ImageService: imageService,
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
	
	// Add multipart form support for file uploads
	srv.AddTransport(transport.MultipartForm{
		MaxUploadSize: 10 << 20, // 10 MB
		MaxMemory:     10 << 20, // 10 MB
	})

	client := measurements.New()

	return requestinfo.Handler()(logger.Handler()(metrics.Handler(client)(srv)))
}
