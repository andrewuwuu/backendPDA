package config

import (
	"strings"
	"testing"
)

func TestDatabaseConfigDSNUsesJakartaLocation(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "db.example",
		Port:     "3306",
		User:     "user",
		Password: "pass",
		Name:     "pda_monitor",
	}

	dsn := cfg.DSN()
	if !strings.Contains(dsn, "parseTime=true") {
		t.Fatalf("expected parseTime=true in DSN, got %q", dsn)
	}
	if !strings.Contains(dsn, "loc=Asia%2FJakarta") {
		t.Fatalf("expected Asia/Jakarta location in DSN, got %q", dsn)
	}
	if strings.Contains(dsn, "loc=Local") {
		t.Fatalf("did not expect loc=Local in DSN, got %q", dsn)
	}
}
