//go:build integration

// Package follows holds the end-to-end tests for the follow graph.
//
// They boot the real GraphQL handler in-process over the real database and
// drive it over HTTP, exactly as the Cosmo router does: identity arrives as
// the x-user-id header the gateway sets. Unlike the package above, this one
// does not start the full server, which needs the key-management service and
// MinIO to come up; nothing in the follow graph does.
package follows

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/weeb-vip/user-service/http/handlers"
	"github.com/weeb-vip/user-service/internal/db"
	"github.com/weeb-vip/user-service/internal/logger"
	"gorm.io/gorm"
)

var (
	server   *httptest.Server
	database *gorm.DB
)

func TestMain(m *testing.M) {
	// The same variables the Makefile and CI set; defaults match
	// docker-compose.yml so `make test-integration` works out of the box.
	defaults := map[string]string{
		"APP_ENV": "dev", "DBHOST": "localhost", "DBPORT": "5432", "DBUSER": "postgres",
		"DBPASSWORD": "postgres", "DBNAME": "weeb", "DBSSL": "disable",
		"DBMIGRATIONTABLE": "__migrations_user",
	}
	for k, v := range defaults {
		if os.Getenv(k) == "" {
			os.Setenv(k, v)
		}
	}
	logger.Logger(logger.WithServerName("user-service-test"), logger.WithVersion("test"), logger.WithEnvironment("test"))

	database = db.GetDBService().GetDB()
	if err := database.WithContext(context.Background()).Exec("SELECT 1 FROM user_follows LIMIT 1").Error; err != nil {
		fmt.Println("user_follows table missing; run the migrations first:", err)
		os.Exit(1)
	}

	// No tokenizer: nothing in the follow graph signs tokens.
	server = httptest.NewServer(handlers.BuildRootHandler(nil))
	code := m.Run()
	server.Close()
	os.Exit(code)
}
