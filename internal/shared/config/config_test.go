package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFile(t *testing.T) {
	path := writeEnvFile(t, `
HTTP_ADDR=:9090
DATABASE_URL=postgres://postgres:postgres@localhost:5432/parties?sslmode=disable
RUN_MIGRATIONS=false
LOG_LEVEL=debug
`)

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":9090")
	}
	if cfg.DatabaseURL != "postgres://postgres:postgres@localhost:5432/parties?sslmode=disable" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.RunMigrations {
		t.Fatal("RunMigrations = true, want false")
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
}

func TestLoadFileRequiresDatabaseURL(t *testing.T) {
	path := writeEnvFile(t, `HTTP_ADDR=:9090`)

	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("LoadFile() error = nil, want error")
	}
}

func TestLoadFilePrefersEnvironment(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://env:env@localhost:5432/env?sslmode=disable")

	path := writeEnvFile(t, `
DATABASE_URL=postgres://file:file@localhost:5432/file?sslmode=disable
RUN_MIGRATIONS=true
`)

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if cfg.DatabaseURL != "postgres://env:env@localhost:5432/env?sslmode=disable" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
}

func TestLoadFileRejectsInvalidBool(t *testing.T) {
	path := writeEnvFile(t, `
DATABASE_URL=postgres://postgres:postgres@localhost:5432/parties?sslmode=disable
RUN_MIGRATIONS=sometimes
`)

	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("LoadFile() error = nil, want error")
	}
}

func writeEnvFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	return path
}
