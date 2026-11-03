package testmigration

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
)

type fakeRunner struct {
	called bool
	err    error
}

func (r *fakeRunner) Up() error {
	r.called = true
	return r.err
}

func validEnv() map[string]string {
	return map[string]string{
		"APP_ENV":           "test",
		"TEST_DATABASE_URL": "movara_migrator:secret@tcp(127.0.0.1:3306)/movara_chat_test",
		"DATABASE_URL":      "app:secret@tcp(127.0.0.1:3306)/movara",
	}
}

func TestRunValidatesBeforeConnectorAndRunner(t *testing.T) {
	connectorCalled := false
	runnerCalled := false
	err := Run(validEnv(), func(string) (*sql.DB, error) {
		connectorCalled = true
		return nil, nil
	}, func(*sql.DB) (Runner, func(), error) {
		runnerCalled = true
		return &fakeRunner{}, nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !connectorCalled || !runnerCalled {
		t.Fatalf("expected connector and runner to be called: connector=%v runner=%v", connectorCalled, runnerCalled)
	}
}

func TestRunRejectsInvalidConfigBeforeConnectorAndRunner(t *testing.T) {
	cases := []map[string]string{
		{"APP_ENV": "development", "TEST_DATABASE_URL": validEnv()["TEST_DATABASE_URL"]},
		{"APP_ENV": "test", "DATABASE_URL": validEnv()["DATABASE_URL"]},
		{"APP_ENV": "test", "TEST_DATABASE_URL": "movara_migrator:secret@tcp(127.0.0.1:3306)/movara"},
		{"APP_ENV": "test", "TEST_DATABASE_URL": "movara_test:secret@tcp(127.0.0.1:3306)/movara_chat_test"},
		{"APP_ENV": "test", "TEST_DATABASE_URL": "movara_migrator:secret@tcp(192.0.2.10:3306)/movara_chat_test"},
		{"APP_ENV": "test", "TEST_DATABASE_URL": "movara_migrator:secret@tcp(127.0.0.1:3307)/movara_chat_test"},
	}
	for _, env := range cases {
		connectorCalled := false
		runnerCalled := false
		err := Run(env, func(string) (*sql.DB, error) {
			connectorCalled = true
			return nil, nil
		}, func(*sql.DB) (Runner, func(), error) {
			runnerCalled = true
			return &fakeRunner{}, nil, nil
		})
		if err == nil || connectorCalled || runnerCalled {
			t.Fatalf("invalid config was not fail-closed: env=%#v err=%v connector=%v runner=%v", env, err, connectorCalled, runnerCalled)
		}
	}
}

func TestRunDoesNotExposeCredentials(t *testing.T) {
	password := "secret-not-for-output"
	err := Run(map[string]string{
		"APP_ENV":           "test",
		"TEST_DATABASE_URL": "movara_migrator:" + password + "@tcp(127.0.0.1:3306)/movara",
	}, func(string) (*sql.DB, error) { return nil, errors.New("connector detail") }, func(*sql.DB) (Runner, func(), error) {
		return &fakeRunner{}, nil, nil
	})
	if err == nil || strings.Contains(err.Error(), password) || strings.Contains(err.Error(), "movara_migrator") {
		t.Fatalf("error exposed sensitive data: %v", err)
	}
}

func TestRunHandlesConnectorAndRunnerFailures(t *testing.T) {
	if err := Run(validEnv(), func(string) (*sql.DB, error) {
		return nil, errors.New("database detail")
	}, func(*sql.DB) (Runner, func(), error) { return &fakeRunner{}, nil, nil }); err == nil || err.Error() != "test migration connector failed" {
		t.Fatalf("unexpected connector error: %v", err)
	}

	if err := Run(validEnv(), func(string) (*sql.DB, error) { return nil, nil }, func(*sql.DB) (Runner, func(), error) {
		return &fakeRunner{err: errors.New("migration detail")}, nil, nil
	}); err == nil || err.Error() != "test migration failed" {
		t.Fatalf("unexpected runner error: %v", err)
	}
}

func TestRunDoesNotCallUpWhenRunnerFactoryFails(t *testing.T) {
	fake := &fakeRunner{}
	password := "factory-secret-not-for-output"
	err := Run(validEnv(), func(string) (*sql.DB, error) {
		return nil, nil
	}, func(*sql.DB) (Runner, func(), error) {
		return fake, nil, errors.New("runner factory failed for " + password)
	})
	if err == nil {
		t.Fatal("expected runner factory error")
	}
	if fake.called {
		t.Fatal("runner Up must not be called when factory initialization fails")
	}
	if strings.Contains(err.Error(), password) || strings.Contains(err.Error(), "movara_migrator") {
		t.Fatalf("error exposed credential or username: %v", err)
	}
}

func TestRunPassesNormalizedValidatedDSNToConnector(t *testing.T) {
	rawDSN := "  movara_migrator:secret@tcp(127.0.0.1:3306)/movara_chat_test  "
	env := validEnv()
	env["TEST_DATABASE_URL"] = rawDSN
	var receivedDSN string
	err := Run(env, func(dsn string) (*sql.DB, error) {
		receivedDSN = dsn
		return nil, nil
	}, func(*sql.DB) (Runner, func(), error) {
		return &fakeRunner{}, nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(rawDSN)
	if receivedDSN != want {
		t.Fatalf("connector received %q, want normalized DSN %q", receivedDSN, want)
	}
}

func TestRunHandlesNoChangeAndDirtyWithoutForce(t *testing.T) {
	if err := Run(validEnv(), func(string) (*sql.DB, error) { return nil, nil }, func(*sql.DB) (Runner, func(), error) {
		return &fakeRunner{err: migrate.ErrNoChange}, nil, nil
	}); err != nil {
		t.Fatalf("no change should be successful: %v", err)
	}

	if err := Run(validEnv(), func(string) (*sql.DB, error) { return nil, nil }, func(*sql.DB) (Runner, func(), error) {
		return &fakeRunner{err: migrate.ErrDirty{Version: 1}}, nil, nil
	}); err == nil || err.Error() != "test migration database is dirty; manual review required" {
		t.Fatalf("dirty state was not blocked: %v", err)
	}
}
