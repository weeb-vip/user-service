package migrations

import (
	"embed"
	"net/http"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"

	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	"gorm.io/gorm"
)

var (
	//go:embed postgres
	migrations embed.FS
)

func New(db *gorm.DB, migrationTableName string) (*migrate.Migrate, error) {
	dbDriver, err := getDBDriver(db, migrationTableName)
	if err != nil {
		return nil, err
	}

	source, err := httpfs.New(http.FS(migrations), "postgres")
	if err != nil {
		return nil, err
	}

	return migrate.NewWithInstance("httpfs", source, "postgres", dbDriver)
}

func getDBDriver(db *gorm.DB, migrationTableName string) (database.Driver, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	return postgres.WithInstance(sqlDB, &postgres.Config{
		MigrationsTable: migrationTableName,
	})
}
