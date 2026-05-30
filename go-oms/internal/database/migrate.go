package database

import (
	"embed"
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func RunDatabaseMigrations(databaseURL string) error {
	sourceDriver, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create embedded iofs driver: %w", err)
	}

	migrationEngine, err := migrate.NewWithSourceInstance("iofs", sourceDriver, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to initialize migration engine: %w", err)
	}
	defer migrationEngine.Close()

	log.Println("Checking database schema version and running migrations...")
	if err := migrationEngine.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("Database schema is already fully up-to-date.")
			return nil
		}
		return fmt.Errorf("critical database migration failed: %w", err)
	}

	log.Println("Database schema successfully migrated to the latest version.")
	return nil
}
