package config

// ApprovedTestDatabasePolicy returns the reviewed, non-secret local test target.
// It is intentionally not used by application startup and contains no password/DSN.
// The database and account must still be created and privilege-checked separately.
func ApprovedTestDatabasePolicy() TestDatabasePolicy {
	return TestDatabasePolicy{
		AllowedDatabases: map[string]struct{}{"movara_chat_test": {}},
		AllowedHosts:     map[string]struct{}{"127.0.0.1": {}},
		AllowedPorts:     map[string]struct{}{"3306": {}},
		AllowedUsers:     map[string]struct{}{"movara_test": {}},
	}
}

// ApprovedTestMigrationDatabasePolicy returns the reviewed target for schema
// migration. It is intentionally separate from the runtime test account.
func ApprovedTestMigrationDatabasePolicy() TestDatabasePolicy {
	return TestDatabasePolicy{
		AllowedDatabases: map[string]struct{}{"movara_chat_test": {}},
		AllowedHosts:     map[string]struct{}{"127.0.0.1": {}},
		AllowedPorts:     map[string]struct{}{"3306": {}},
		AllowedUsers:     map[string]struct{}{"movara_migrator": {}},
	}
}
