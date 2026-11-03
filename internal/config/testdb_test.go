package config

import (
	"strings"
	"testing"
)

var testDBPolicy = TestDatabasePolicy{
	AllowedDatabases: map[string]struct{}{"movara_test": {}},
	AllowedHosts:     map[string]struct{}{"127.0.0.1": {}},
	AllowedPorts:     map[string]struct{}{"3307": {}},
	AllowedUsers:     map[string]struct{}{"test_user": {}},
}

func testDBEnv(dsn string) map[string]string {
	return map[string]string{"APP_ENV": "test", "TEST_DATABASE_URL": dsn}
}

func TestValidateTestDatabaseRejectsMissingOrWrongEnvironment(t *testing.T) {
	if _, err := ValidateTestDatabase(map[string]string{}, testDBPolicy); err == nil {
		t.Fatal("expected missing environment to be rejected")
	}
	if _, err := ValidateTestDatabase(map[string]string{"APP_ENV": "development", "TEST_DATABASE_URL": "user:password@tcp(127.0.0.1:3307)/movara_test"}, testDBPolicy); err == nil {
		t.Fatal("expected non-test environment to be rejected")
	}
}

func TestApprovedTestDatabasePolicyUsesOnlyApprovedNonSecretTarget(t *testing.T) {
	policy := ApprovedTestDatabasePolicy()
	if _, ok := policy.AllowedDatabases["movara_chat_test"]; !ok {
		t.Fatal("approved database target is missing")
	}
	if _, ok := policy.AllowedHosts["127.0.0.1"]; !ok {
		t.Fatal("approved host target is missing")
	}
	if _, ok := policy.AllowedPorts["3306"]; !ok {
		t.Fatal("approved port target is missing")
	}
	if _, ok := policy.AllowedUsers["movara_test"]; !ok {
		t.Fatal("approved user target is missing")
	}
	if _, err := ValidateTestDatabase(testDBEnv("test_user:test_password@tcp(127.0.0.1:3307)/movara_test"), policy); err == nil {
		t.Fatal("expected unapproved DSN target to be rejected")
	}
}

func TestApprovedTestMigrationDatabasePolicyUsesMigratorOnly(t *testing.T) {
	policy := ApprovedTestMigrationDatabasePolicy()
	if _, ok := policy.AllowedDatabases["movara_chat_test"]; !ok {
		t.Fatal("approved migration database target is missing")
	}
	if _, ok := policy.AllowedHosts["127.0.0.1"]; !ok {
		t.Fatal("approved migration host target is missing")
	}
	if _, ok := policy.AllowedPorts["3306"]; !ok {
		t.Fatal("approved migration port target is missing")
	}
	if _, ok := policy.AllowedUsers["movara_migrator"]; !ok {
		t.Fatal("approved migration user is missing")
	}
	if _, ok := policy.AllowedUsers["movara_test"]; ok {
		t.Fatal("runtime user must not be approved for migration")
	}
}

func TestValidateTestMigrationDatabaseRejectsUnsafeTargets(t *testing.T) {
	policy := ApprovedTestMigrationDatabasePolicy()
	cases := []string{
		"test_user:secret@tcp(127.0.0.1:3306)/movara",
		"test_user:secret@tcp(127.0.0.1:3306)/other_test",
		"test_user:secret@tcp(192.0.2.10:3306)/movara_chat_test",
		"test_user:secret@tcp(127.0.0.1:3307)/movara_chat_test",
		"movara_test:secret@tcp(127.0.0.1:3306)/movara_chat_test",
		"movara_migrator:secret@unix(/var/run/mysqld/mysqld.sock)/movara_chat_test",
	}
	for _, dsn := range cases {
		if _, err := ValidateTestDatabase(testDBEnv(dsn), policy); err == nil {
			t.Errorf("expected unsafe migration target to be rejected: %q", dsn)
		}
	}
}

func TestValidateTestMigrationDatabaseRequiresTestEnvironmentAndURL(t *testing.T) {
	policy := ApprovedTestMigrationDatabasePolicy()
	validDSN := "movara_migrator:secret@tcp(127.0.0.1:3306)/movara_chat_test"

	for _, env := range []map[string]string{
		{},
		{"APP_ENV": "development", "TEST_DATABASE_URL": validDSN},
		{"APP_ENV": "test"},
		{"APP_ENV": "test", "TEST_DATABASE_URL": "", "DATABASE_URL": "user:secret@tcp(127.0.0.1:3306)/movara"},
	} {
		if _, err := ValidateTestDatabase(env, policy); err == nil {
			t.Errorf("expected environment to be rejected: %#v", env)
		}
	}
}

func TestValidateTestMigrationDatabaseErrorsDoNotEchoCredentials(t *testing.T) {
	password := "migration-secret-not-for-output"
	username := "movara_migrator"
	_, err := ValidateTestDatabase(testDBEnv(username+":"+password+"@tcp(127.0.0.1:3306)/movara"), ApprovedTestMigrationDatabasePolicy())
	if err == nil {
		t.Fatal("expected application database target to be rejected")
	}
	if message := err.Error(); message == "" || containsAny(message, password, username, "movara") {
		t.Fatalf("error must not expose credentials or target details: %q", message)
	}
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func TestValidateTestDatabaseRejectsInvalidPolicy(t *testing.T) {
	bad := testDBPolicy
	bad.AllowedUsers = nil
	if _, err := ValidateTestDatabase(testDBEnv("test_user:test_password@tcp(127.0.0.1:3307)/movara_test"), bad); err == nil {
		t.Fatal("expected incomplete policy to be rejected")
	}
	bad = testDBPolicy
	bad.AllowedPorts = map[string]struct{}{"3306-3307": {}}
	if _, err := ValidateTestDatabase(testDBEnv("test_user:test_password@tcp(127.0.0.1:3307)/movara_test"), bad); err == nil {
		t.Fatal("expected malformed policy port to be rejected")
	}
}

func TestValidateTestDatabaseDoesNotFallbackToApplicationDatabase(t *testing.T) {
	env := map[string]string{"APP_ENV": "test", "DATABASE_URL": "user:password@tcp(127.0.0.1:3307)/movara_test"}
	if _, err := ValidateTestDatabase(env, testDBPolicy); err == nil {
		t.Fatal("expected missing TEST_DATABASE_URL to be rejected")
	}
}

func TestValidateTestDatabaseRejectsInvalidTarget(t *testing.T) {
	cases := []string{
		"not-a-dsn",
		"user:password@tcp(127.0.0.1:3307)/movara_development",
		"user:password@tcp(192.0.2.10:3307)/movara_test",
		"user:password@tcp(127.0.0.1:3306)/movara_test",
	}
	for _, dsn := range cases {
		if _, err := ValidateTestDatabase(testDBEnv(dsn), testDBPolicy); err == nil {
			t.Fatalf("expected target to be rejected: %q", dsn)
		}
	}
}

func TestValidateTestDatabaseAcceptsExplicitPolicy(t *testing.T) {
	cfg, err := ValidateTestDatabase(testDBEnv("test_user:test_password@tcp(127.0.0.1:3307)/movara_test"), testDBPolicy)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database != "movara_test" || cfg.Host != "127.0.0.1" || cfg.Port != "3307" || cfg.User != "test_user" {
		t.Fatalf("unexpected parsed target: %#v", cfg)
	}
}

func TestValidateTestDatabaseErrorsDoNotEchoCredentials(t *testing.T) {
	password := "secret-not-for-output"
	_, err := ValidateTestDatabase(testDBEnv("test_user:"+password+"@tcp(127.0.0.1:3306)/movara_test"), testDBPolicy)
	if err == nil {
		t.Fatal("expected invalid port")
	}
	if contains := err.Error(); contains == "" || contains == password || contains == "test_user" || contains == "movara_test" {
		t.Fatalf("error must not expose credential: %q", contains)
	}
}
