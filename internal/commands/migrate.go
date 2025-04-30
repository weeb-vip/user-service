package commands

import (
	"errors"

	"github.com/weeb-vip/user/config"

	"github.com/golang-migrate/migrate/v4"

	"github.com/weeb-vip/user/internal/db"
	"github.com/weeb-vip/user/internal/migrations"

	"github.com/spf13/cobra"
)

func configureMigrateCommand(rootCmd *cobra.Command) {
	dbCommand := &cobra.Command{
		Use:   "db",
		Short: "manage database",
	}
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "migrate database",
		RunE:  migrateDB,
	}
	dbCommand.AddCommand(migrateCmd)

	rootCmd.AddCommand(dbCommand)
}

func migrateDB(cmd *cobra.Command, _ []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	dbService := db.GetDBService()

	migration, err := migrations.New(dbService.GetDB(), cfg.DBConfig.MigrationTableName)

	if err != nil {
		return err
	}

	cmd.Println("Migrating...")

	err = migration.Up()

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
