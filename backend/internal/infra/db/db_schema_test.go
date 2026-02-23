package db

import "testing"

// このテストは旧価格参照経路（product_prices）の残存を検知する回帰テスト。
// 仕様対象: テスト用スキーマ準備後に product_prices が存在しないこと。
// 根拠: 価格決定元を products に一本化し、暫定運用へ戻る回帰を防ぐため。
func TestEnsureSchema_回帰_旧価格参照テーブルproduct_pricesを作成しない(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ensureSchema(t, db)
	resetTables(t, db)

	var tableName *string
	if err := db.QueryRow(`SELECT to_regclass('public.product_prices')::text`).Scan(&tableName); err != nil {
		t.Fatalf("failed to check product_prices existence: %v", err)
	}
	if tableName != nil {
		t.Fatalf("product_prices must not exist, got %s", *tableName)
	}
}
