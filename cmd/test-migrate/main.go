package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"movara/internal/testmigration"
	"movara/migrations"
)

func main() {
	env := map[string]string{
		"APP_ENV":           os.Getenv("APP_ENV"),
		"TEST_DATABASE_URL": os.Getenv("TEST_DATABASE_URL"),
	}
	if err := testmigration.Run(env, func(dsn string) (*sql.DB, error) {
		return sql.Open("mysql", dsn)
	}, newRunner); err != nil {
		log.Fatal(err)
	}
}

func newRunner(db *sql.DB) (testmigration.Runner, func(), error) {
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, nil, err
	}
	databaseDriver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return nil, nil, err
	}
	runner, err := migrate.NewWithInstance("iofs", source, "mysql", databaseDriver)
	if err != nil {
		return nil, nil, err
	}
	return runner, func() {
		_, _ = runner.Close()
	}, nil
}
