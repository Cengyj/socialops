package config

import (
	"net/url"
	"testing"
)

func TestDatabaseConfigDSNEncodesPasswordSpecialCharacters(t *testing.T) {
	cfg := &DatabaseConfig{
		Host:     "db.example.com",
		Port:     5432,
		User:     "socialops",
		Password: `pa ss'word\?&=`,
		DBName:   "socialops",
		SSLMode:  "require",
	}

	parsed := parsePostgresURL(t, cfg.DSN())
	password, ok := parsed.User.Password()

	if !ok {
		t.Fatalf("DSN should include a password, got %q", cfg.DSN())
	}
	if password != cfg.Password {
		t.Fatalf("password round-trip mismatch: got %q, want %q; dsn=%q", password, cfg.Password, cfg.DSN())
	}
	if parsed.User.Username() != cfg.User {
		t.Fatalf("username round-trip mismatch: got %q, want %q", parsed.User.Username(), cfg.User)
	}
	if parsed.Path != "/"+cfg.DBName {
		t.Fatalf("database path mismatch: got %q, want %q", parsed.Path, "/"+cfg.DBName)
	}
	if parsed.Query().Get("sslmode") != cfg.SSLMode {
		t.Fatalf("sslmode mismatch: got %q, want %q; dsn=%q", parsed.Query().Get("sslmode"), cfg.SSLMode, cfg.DSN())
	}
}

func TestDatabaseConfigDSNWithTimezoneEncodesTimezone(t *testing.T) {
	cfg := &DatabaseConfig{
		Host:     "127.0.0.1",
		Port:     5432,
		User:     "postgres",
		Password: "secret",
		DBName:   "socialops",
		SSLMode:  "disable",
	}

	parsed := parsePostgresURL(t, cfg.DSNWithTimezone("Asia/Shanghai"))

	if parsed.Query().Get("TimeZone") != "Asia/Shanghai" {
		t.Fatalf("timezone mismatch: got %q, want Asia/Shanghai; dsn=%q", parsed.Query().Get("TimeZone"), cfg.DSNWithTimezone("Asia/Shanghai"))
	}
}

func parsePostgresURL(t *testing.T, dsn string) *url.URL {
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
