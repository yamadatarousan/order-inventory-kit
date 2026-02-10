package main

import "testing"

func TestLoadConfig_環境変数が未設定ならデフォルト値を使う(t *testing.T) {
	t.Setenv("API_ADDR", "")
	t.Setenv("DATABASE_URL", "")

	cfg := loadConfig()

	if cfg.APIAddr != defaultAPIAddr {
		t.Fatalf("expected default api addr %s, got %s", defaultAPIAddr, cfg.APIAddr)
	}
	if cfg.DatabaseURL != defaultDatabaseURL {
		t.Fatalf("expected default database url %s, got %s", defaultDatabaseURL, cfg.DatabaseURL)
	}
}

func TestLoadConfig_環境変数が設定されていれば優先する(t *testing.T) {
	t.Setenv("API_ADDR", ":9090")
	t.Setenv("DATABASE_URL", "postgres://example")

	cfg := loadConfig()

	if cfg.APIAddr != ":9090" {
		t.Fatalf("expected :9090, got %s", cfg.APIAddr)
	}
	if cfg.DatabaseURL != "postgres://example" {
		t.Fatalf("expected postgres://example, got %s", cfg.DatabaseURL)
	}
}
