package config

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	mysql "github.com/go-sql-driver/mysql"
)

// TestDatabasePolicy sengaja tidak memiliki default deployment target.
// Pemanggil wajib menginjeksikan allowlist yang telah disetujui.
type TestDatabasePolicy struct {
	AllowedDatabases map[string]struct{}
	AllowedHosts     map[string]struct{}
	AllowedPorts     map[string]struct{}
	AllowedUsers     map[string]struct{}
}

type TestDatabaseConfig struct {
	Database string
	Host     string
	Port     string
	User     string
}

// ValidateTestDatabase memvalidasi konfigurasi tanpa membuka koneksi database.
// Ia hanya membaca APP_ENV dan TEST_DATABASE_URL dari env yang diberikan.
func ValidateTestDatabase(env map[string]string, policy TestDatabasePolicy) (TestDatabaseConfig, error) {
	if env["APP_ENV"] != "test" {
		return TestDatabaseConfig{}, errors.New("test database requires APP_ENV=test")
	}
	dsn := strings.TrimSpace(env["TEST_DATABASE_URL"])
	if dsn == "" {
		return TestDatabaseConfig{}, errors.New("TEST_DATABASE_URL is required")
	}
	if err := validateTestDatabasePolicy(policy); err != nil {
		return TestDatabaseConfig{}, err
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return TestDatabaseConfig{}, errors.New("TEST_DATABASE_URL is invalid")
	}
	host, port, err := splitMySQLAddress(cfg.Net, cfg.Addr)
	if err != nil {
		return TestDatabaseConfig{}, errors.New("TEST_DATABASE_URL address is invalid")
	}
	if _, ok := policy.AllowedDatabases[cfg.DBName]; !ok {
		return TestDatabaseConfig{}, errors.New("test database name is not allowed")
	}
	if _, ok := policy.AllowedHosts[host]; !ok {
		return TestDatabaseConfig{}, errors.New("test database host is not allowed")
	}
	if _, ok := policy.AllowedPorts[port]; !ok {
		return TestDatabaseConfig{}, errors.New("test database port is not allowed")
	}
	if _, ok := policy.AllowedUsers[cfg.User]; !ok {
		return TestDatabaseConfig{}, errors.New("test database user is not allowed")
	}
	return TestDatabaseConfig{Database: cfg.DBName, Host: host, Port: port, User: cfg.User}, nil
}

func validateTestDatabasePolicy(policy TestDatabasePolicy) error {
	if len(policy.AllowedDatabases) == 0 || len(policy.AllowedHosts) == 0 || len(policy.AllowedPorts) == 0 || len(policy.AllowedUsers) == 0 {
		return errors.New("test database policy must define database, host, port, and user allowlists")
	}
	for _, values := range []map[string]struct{}{policy.AllowedDatabases, policy.AllowedHosts, policy.AllowedPorts, policy.AllowedUsers} {
		for value := range values {
			if strings.TrimSpace(value) != value || value == "" || strings.ContainsAny(value, "*?\r\n") {
				return errors.New("test database policy contains an invalid allowlist value")
			}
		}
	}
	for port := range policy.AllowedPorts {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return errors.New("test database policy contains an invalid port")
		}
	}
	return nil
}

func splitMySQLAddress(network, address string) (string, string, error) {
	if network == "unix" {
		return "unix", "", fmt.Errorf("unix socket is not allowed by test policy")
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" || port == "" {
		return "", "", fmt.Errorf("invalid address")
	}
	return strings.ToLower(host), port, nil
}
