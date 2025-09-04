package user

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/rs/cors"
	"github.com/weeb-vip/user-service/config"
	"github.com/weeb-vip/user-service/http/handlers"
	"github.com/weeb-vip/user-service/internal/jwt"
	"github.com/weeb-vip/user-service/internal/keypair"
	"github.com/weeb-vip/user-service/internal/publishkey"

	"github.com/99designs/gqlgen/graphql/playground"
)

const minKeyValidityDurationMinutes = 5

func StartServer() error { // nolint
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	rotatingKey, err := getRotatingSigningKey(cfg)
	if err != nil {
		return err
	}

	router := chi.NewRouter()

	router.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8081", "http://localhost:3000"},
		AllowCredentials: true,
		Debug:            false,
	}).Handler)

	router.Handle("/", playground.Handler("GraphQL playground", "/graphql"))
	router.Handle("/graphql", handlers.BuildRootHandler(jwt.New(rotatingKey)))
	router.Handle("/readyz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200) // nolint
	}))
	router.Handle("/livez", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200) // nolint
	}))

	log.Printf("connect to http://localhost:%d/ for GraphQL playground", cfg.APPConfig.Port)

	return http.ListenAndServe(fmt.Sprintf(":%d", cfg.APPConfig.Port), router) // nolint
}

func getRotatingSigningKey(cfg *config.Config) (keypair.RotatingSigningKey, error) {
	rotatingKey, err := keypair.NewSigningKeyRotator(
		publishkey.NewKeyPublisher(
			cfg.APPConfig.InternalGraphQLURL).
			PublishToKeyManagementService)
	if err != nil {
		return nil, err
	}

	requestedDuration := time.Hour * time.Duration(cfg.APPConfig.KeyRollingDurationInHours)
	rotatingKey.RotateInBackground(getMinimumDuration(requestedDuration, time.Minute*minKeyValidityDurationMinutes))

	return rotatingKey, nil
}

func getMinimumDuration(askedDuration time.Duration, minimumDuration time.Duration) time.Duration {
	if askedDuration < minimumDuration {
		return minimumDuration
	}

	return askedDuration
}
