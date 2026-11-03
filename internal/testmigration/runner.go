package testmigration

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"movara/internal/config"
)

// Connector is deliberately injected so validation can be tested without a
// database connection.
type Connector func(dsn string) (*sql.DB, error)

// Runner is the smallest operation needed by this entry point.
type Runner interface {
	Up() error
}

// RunnerFactory is called only after the target has passed policy validation.
type RunnerFactory func(db *sql.DB) (Runner, func(), error)

// Run validates the exact normalized DSN passed to the connector. It never
// reads DATABASE_URL and never starts the application lifecycle.
func Run(env map[string]string, connector Connector, factory RunnerFactory) error {
	dsn := strings.TrimSpace(env["TEST_DATABASE_URL"])
	validatedEnv := map[string]string{
		"APP_ENV":           env["APP_ENV"],
		"TEST_DATABASE_URL": dsn,
	}
	if _, err := config.ValidateTestDatabase(validatedEnv, config.ApprovedTestMigrationDatabasePolicy()); err != nil {
		return err
	}

	db, err := connector(dsn)
	if err != nil {
		return errors.New("test migration connector failed")
	}
	if db != nil {
		defer db.Close()
	}

	runner, cleanup, err := factory(db)
	if err != nil {
		return errors.New("test migration runner initialization failed")
	}
	if cleanup != nil {
		defer cleanup()
	}

	if err := runner.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		var dirty migrate.ErrDirty
		if errors.As(err, &dirty) {
			return errors.New("test migration database is dirty; manual review required")
		}
		return errors.New("test migration failed")
	}
	return nil
}
