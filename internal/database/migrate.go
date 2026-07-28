package database

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	migratedatabase "github.com/golang-migrate/migrate/v4/database"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	migratesqlite "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

//go:embed migrations/postgresql/*.sql migrations/sqlite/*.sql
var migrationFiles embed.FS

func migrateSchema(backend Backend, dsn string) error {
	sqlDriver, migrationDir, migrationDriverName := migrationSettings(backend)

	db, err := sql.Open(sqlDriver, dsn)
	if err != nil {
		return err
	}

	databaseDriver, err := newMigrationDatabaseDriver(backend, db)
	if err != nil {
		db.Close()
		return err
	}

	sourceDriver, err := iofs.New(migrationFiles, migrationDir)
	if err != nil {
		databaseDriver.Close()
		return err
	}

	migrator, err := migrate.NewWithInstance(
		"iofs",
		sourceDriver,
		migrationDriverName,
		databaseDriver,
	)
	if err != nil {
		sourceDriver.Close()
		databaseDriver.Close()
		return err
	}

	migrationErr := migrator.Up()
	sourceErr, databaseErr := migrator.Close()

	if migrationErr != nil && !errors.Is(migrationErr, migrate.ErrNoChange) {
		return migrationErr
	}
	if sourceErr != nil {
		return fmt.Errorf("close migration source: %w", sourceErr)
	}
	if databaseErr != nil {
		return fmt.Errorf("close migration database: %w", databaseErr)
	}
	return nil
}

func migrationSettings(backend Backend) (
	sqlDriver string,
	migrationDir string,
	migrationDriverName string,
) {
	if backend == BackendPostgreSQL {
		return "pgx", "migrations/postgresql", "postgres"
	}
	return "sqlite", "migrations/sqlite", "sqlite"
}

func newMigrationDatabaseDriver(
	backend Backend,
	db *sql.DB,
) (migratedatabase.Driver, error) {
	if backend == BackendPostgreSQL {
		return migratepostgres.WithInstance(db, &migratepostgres.Config{})
	}
	return migratesqlite.WithInstance(db, &migratesqlite.Config{})
}
