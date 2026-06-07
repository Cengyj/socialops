package setup

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDecideAdminBootstrap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		totalUsers int64
		adminUsers int64
		should     bool
		reason     string
	}{
		{
			name:       "empty database should create admin",
			totalUsers: 0,
			adminUsers: 0,
			should:     true,
			reason:     adminBootstrapReasonEmptyDatabase,
		},
		{
			name:       "admin exists should skip",
			totalUsers: 10,
			adminUsers: 1,
			should:     false,
			reason:     adminBootstrapReasonAdminExists,
		},
		{
			name:       "users exist without admin should skip",
			totalUsers: 5,
			adminUsers: 0,
			should:     false,
			reason:     adminBootstrapReasonUsersExistWithoutAdmin,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := decideAdminBootstrap(tc.totalUsers, tc.adminUsers)
			if got.shouldCreate != tc.should {
				t.Fatalf("shouldCreate=%v, want %v", got.shouldCreate, tc.should)
			}
			if got.reason != tc.reason {
				t.Fatalf("reason=%q, want %q", got.reason, tc.reason)
			}
		})
	}
}

func TestSetupDefaultAdminConcurrency(t *testing.T) {
	t.Run("simple mode admin uses higher concurrency", func(t *testing.T) {
		t.Setenv("RUN_MODE", "simple")
		if got := setupDefaultAdminConcurrency(); got != simpleModeAdminConcurrency {
			t.Fatalf("setupDefaultAdminConcurrency()=%d, want %d", got, simpleModeAdminConcurrency)
		}
	})

	t.Run("standard mode keeps existing default", func(t *testing.T) {
		t.Setenv("RUN_MODE", "standard")
		if got := setupDefaultAdminConcurrency(); got != defaultUserConcurrency {
			t.Fatalf("setupDefaultAdminConcurrency()=%d, want %d", got, defaultUserConcurrency)
		}
	})
}

func TestWriteConfigFileKeepsDefaultUserConcurrency(t *testing.T) {
	t.Setenv("RUN_MODE", "simple")
	t.Setenv("DATA_DIR", t.TempDir())

	if err := writeConfigFile(&SetupConfig{}); err != nil {
		t.Fatalf("writeConfigFile() error = %v", err)
	}

	data, err := os.ReadFile(GetConfigFilePath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(data), "user_concurrency: 5") {
		t.Fatalf("config missing default user concurrency, got:\n%s", string(data))
	}
}

func TestWriteConfigFileCreatesConfiguredDataDir(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "nested", "data")
	t.Setenv("DATA_DIR", dataDir)

	if err := writeConfigFile(&SetupConfig{}); err != nil {
		t.Fatalf("writeConfigFile() error = %v", err)
	}

	info, err := os.Stat(dataDir)
	if err != nil {
		t.Fatalf("expected DATA_DIR to be created: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("DATA_DIR path should be a directory")
	}
	if _, err := os.Stat(GetConfigFilePath()); err != nil {
		t.Fatalf("expected config file to be written: %v", err)
	}
}

func TestCreateInstallLockCreatesConfiguredDataDir(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "nested", "data")
	t.Setenv("DATA_DIR", dataDir)

	if err := createInstallLock(); err != nil {
		t.Fatalf("createInstallLock() error = %v", err)
	}

	data, err := os.ReadFile(GetInstallLockPath())
	if err != nil {
		t.Fatalf("expected install lock to be written: %v", err)
	}
	if !strings.Contains(string(data), "installed_at=") {
		t.Fatalf("install lock missing installed_at marker, got %q", string(data))
	}
}

func TestDatabaseConnectionDSNsUseMaintenanceDatabaseBeforeTarget(t *testing.T) {
	cfg := &DatabaseConfig{
		Host:     "127.0.0.1",
		Port:     5432,
		User:     "postgres",
		Password: "secret",
		DBName:   "socialops",
		SSLMode:  "disable",
	}

	maintenanceDSN, targetDSN := databaseConnectionDSNs(cfg)
	maintenanceURL := parseSetupPostgresURL(t, maintenanceDSN)
	targetURL := parseSetupPostgresURL(t, targetDSN)

	if maintenanceURL.Path != "/postgres" {
		t.Fatalf("maintenance DSN should connect to postgres database, got %q", maintenanceDSN)
	}
	if maintenanceURL.Path == "/socialops" {
		t.Fatalf("maintenance DSN must not point at target database before it exists, got %q", maintenanceDSN)
	}
	if targetURL.Path != "/socialops" {
		t.Fatalf("target DSN should connect to requested database, got %q", targetDSN)
	}
}

func TestDatabaseConnectionDSNsEncodePasswordSpecialCharacters(t *testing.T) {
	cfg := &DatabaseConfig{
		Host:     "db.example.com",
		Port:     5432,
		User:     "socialops",
		Password: `pa ss'word\?&=`,
		DBName:   "socialops",
		SSLMode:  "require",
	}

	_, targetDSN := databaseConnectionDSNs(cfg)
	targetURL := parseSetupPostgresURL(t, targetDSN)
	password, ok := targetURL.User.Password()

	if !ok {
		t.Fatalf("target DSN should include a password, got %q", targetDSN)
	}
	if password != cfg.Password {
		t.Fatalf("password round-trip mismatch: got %q, want %q; dsn=%q", password, cfg.Password, targetDSN)
	}
	if targetURL.User.Username() != cfg.User {
		t.Fatalf("username round-trip mismatch: got %q, want %q", targetURL.User.Username(), cfg.User)
	}
	if targetURL.Query().Get("sslmode") != cfg.SSLMode {
		t.Fatalf("sslmode mismatch: got %q, want %q; dsn=%q", targetURL.Query().Get("sslmode"), cfg.SSLMode, targetDSN)
	}
}

func TestTestDatabaseConnectionRejectsUnsafeDBNameBeforeNetwork(t *testing.T) {
	cfg := &DatabaseConfig{
		Host:     "203.0.113.1",
		Port:     5432,
		User:     "postgres",
		Password: "secret",
		DBName:   "socialops;DROP_TABLE_users",
		SSLMode:  "disable",
	}

	startedAt := time.Now()
	err := TestDatabaseConnection(cfg)
	elapsed := time.Since(startedAt)

	if err == nil {
		t.Fatal("TestDatabaseConnection() expected an error for unsafe database name")
	}
	if !strings.Contains(err.Error(), "invalid database name") {
		t.Fatalf("error should reject the database name before connecting, got %v", err)
	}
	if elapsed > time.Second {
		t.Fatalf("unsafe database name should be rejected before network access, took %s", elapsed)
	}
}

func parseSetupPostgresURL(t *testing.T, dsn string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse postgres DSN %q: %v", dsn, err)
	}
	if parsed.Scheme != "postgres" {
		t.Fatalf("postgres DSN should use postgres URL syntax, got %q", dsn)
	}
	return parsed
}
