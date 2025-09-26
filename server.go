package user

import (
	"context"
	"fmt"
	"github.com/weeb-vip/user-service/internal/logger"
	"github.com/weeb-vip/user-service/metrics"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"github.com/weeb-vip/user-service/config"
	"github.com/weeb-vip/user-service/http/handlers"
	"github.com/weeb-vip/user-service/http/middleware"
	"github.com/weeb-vip/user-service/internal/jwt"
	"github.com/weeb-vip/user-service/internal/keypair"
	"github.com/weeb-vip/user-service/internal/publishkey"

	"github.com/99designs/gqlgen/graphql/playground"
)

const minKeyValidityDurationMinutes = 5

func StartServer() error { // nolint
	return StartServerWithContext(context.Background())
}

func StartServerWithContext(ctx context.Context) error { // nolint
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	log := logger.FromCtx(ctx)
	log.Info().Msg("Starting server...")

	// Initialize metrics
	log.Info().Msg("Initializing metrics...")
	_ = metrics.GetAppMetrics() // Initialize the metrics singleton
	log.Info().Msg("Metrics initialized successfully")

	log.Info().Msg("Loading keys...")
	rotatingKey, err := getRotatingSigningKey(cfg, ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load keys")
		return err
	}

	log.Info().Msg("Keys loaded successfully")

	router := chi.NewRouter()

	// Add tracing middleware first to capture all requests
	router.Use(middleware.TracingMiddleware())

	// Add gzip compression middleware
	router.Use(middleware.GzipMiddleware())

	router.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8081", "http://localhost:3000"},
		AllowCredentials: true,
		Debug:            false,
	}).Handler)

	router.Handle("/", playground.Handler("GraphQL playground", "/graphql"))
	router.Handle("/graphql", handlers.BuildRootHandler(jwt.New(rotatingKey)))
	router.Handle("/metrics", promhttp.Handler())
	router.Handle("/readyz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200) // nolint
	}))
	router.Handle("/livez", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200) // nolint
	}))

	log.Info().Int("port", cfg.APPConfig.Port).Msg("Connect to GraphQL playground")

	return http.ListenAndServe(fmt.Sprintf(":%d", cfg.APPConfig.Port), router) // nolint
}

func getRotatingSigningKey(cfg *config.Config, ctx context.Context) (keypair.RotatingSigningKey, error) {
	log := logger.FromCtx(ctx)

	rotatingKey, err := keypair.NewSigningKeyRotator(
		publishkey.NewKeyPublisher(
			cfg.APPConfig.InternalGraphQLURL).
			PublishToKeyManagementService)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create signing key rotator")
		return nil, err
	}

	requestedDuration := time.Hour * time.Duration(cfg.APPConfig.KeyRollingDurationInHours)
	actualDuration := getMinimumDuration(requestedDuration, time.Minute*minKeyValidityDurationMinutes)

	log.Info().Dur("duration", actualDuration).Msg("Starting key rotation in background")
	rotatingKey.RotateInBackground(actualDuration)

	return rotatingKey, nil
}

func getMinimumDuration(askedDuration time.Duration, minimumDuration time.Duration) time.Duration {
	if askedDuration < minimumDuration {
		return minimumDuration
	}

	return askedDuration
}
