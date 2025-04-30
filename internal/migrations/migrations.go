package migrations

import (
	"embed"
	"net/http"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"

	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	"gorm.io/gorm"
)

var (
	//go:embed scripts
	migrations embed.FS
)

func New(db *gorm.DB, migrationTableName string) (*migrate.Migrate, error) {
	dbDriver, err := getDBDriver(db, migrationTableName)
	if err != nil {
		return nil, err
	}

	source, err := httpfs.New(http.FS(migrations), "scripts")
	if err != nil {
		return nil, err
	}

	return migrate.NewWithInstance("httpfs", source, "mysql", dbDriver)
}

func getDBDriver(db *gorm.DB, migrationTableName string) (database.Driver, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	return mysql.WithInstance(sqlDB, &mysql.Config{
		MigrationsTable: migrationTableName,
	})
}
