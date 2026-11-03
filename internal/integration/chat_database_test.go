//go:build integration

package integration

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"movara/internal/config"
)

// This test is intentionally excluded from the default test suite. It is an
// explicit, fail-closed entry point for a separately provisioned test DB.
func TestChatDatabaseTargetIsExplicitAndReachable(t *testing.T) {
	if os.Getenv("APP_ENV") != "test" {
		t.Fatal("integration tests require APP_ENV=test")
	}
	policy := config.ApprovedTestDatabasePolicy()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("integration tests require TEST_DATABASE_URL")
	}
	if _, err := config.ValidateTestDatabase(map[string]string{
		"APP_ENV":            os.Getenv("APP_ENV"),
		"TEST_DATABASE_URL": dsn,
	}, policy); err != nil {
		t.Fatalf("test database target rejected: %v", err)
	}

	// The DSN is opened only after policy validation. This stage does not run
	// this test; migration and schema checks remain explicit operator actions.
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer db.Close()
	if err := db.PingContext(t.Context()); err != nil {
		t.Fatalf("ping test database: %v", err)
	}
}
