package main

import "testing"

// このテストは API起動設定の最小仕様を固定する。
// 仕様対象: 環境変数未設定時のデフォルト値適用と、設定時の上書き優先。
// 根拠: 実行環境差異で起動設定が意図せず変化しないようにするため。
func TestLoadConfig_環境変数が未設定ならデフォルト値を使う(t *testing.T) {
	t.Setenv("API_ADDR", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("CORS_ALLOW_ORIGIN", "")

	cfg := loadConfig()

	if cfg.APIAddr != defaultAPIAddr {
		t.Fatalf("expected default api addr %s, got %s", defaultAPIAddr, cfg.APIAddr)
	}
	if cfg.DatabaseURL != defaultDatabaseURL {
		t.Fatalf("expected default database url %s, got %s", defaultDatabaseURL, cfg.DatabaseURL)
	}
	if cfg.CORSAllowOrigin != defaultCORSAllowOrigin {
		t.Fatalf("expected default cors allow origin %s, got %s", defaultCORSAllowOrigin, cfg.CORSAllowOrigin)
	}
}

func TestLoadConfig_環境変数が設定されていれば優先する(t *testing.T) {
	t.Setenv("API_ADDR", ":9090")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("CORS_ALLOW_ORIGIN", "http://localhost:3000")

	cfg := loadConfig()

	if cfg.APIAddr != ":9090" {
		t.Fatalf("expected :9090, got %s", cfg.APIAddr)
	}
	if cfg.DatabaseURL != "postgres://example" {
		t.Fatalf("expected postgres://example, got %s", cfg.DatabaseURL)
	}
	if cfg.CORSAllowOrigin != "http://localhost:3000" {
		t.Fatalf("expected cors origin http://localhost:3000, got %s", cfg.CORSAllowOrigin)
	}
}

func TestParseAllowedOrigins_カンマ区切りを分解できる(t *testing.T) {
	allowed := parseAllowedOrigins("http://localhost:5173, http://localhost:3000")
	if _, ok := allowed["http://localhost:5173"]; !ok {
		t.Fatalf("expected localhost:5173 to be allowed")
	}
	if _, ok := allowed["http://localhost:3000"]; !ok {
		t.Fatalf("expected localhost:3000 to be allowed")
	}
}

func TestIsAllowedOrigin_ワイルドカード指定で許可する(t *testing.T) {
	allowed := parseAllowedOrigins("*")
	if !isAllowedOrigin("http://localhost:5173", allowed) {
		t.Fatalf("expected wildcard to allow any origin")
	}
}
