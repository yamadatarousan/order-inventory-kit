package db

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// このテスト補助は DBリポジトリテストの実行前提を固定する。
// 対象: テスト用DB接続、スキーマ準備、テーブル初期化。
// 根拠: テストごとの前提差異をなくし、失敗原因を実装差分に限定するため。
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		dsn = "postgres://order_inventory:order_inventory@localhost:5434/order_inventory_test?sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping db: %v", err)
	}
	return db
}

func resetTables(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		TRUNCATE TABLE payments, order_items, orders, inventory RESTART IDENTITY CASCADE;
	`)
	if err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}
}

func ensureSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS orders (
		  id TEXT PRIMARY KEY,
		  customer_id TEXT,
		  status TEXT NOT NULL,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS order_items (
		  order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
		  sku TEXT NOT NULL,
		  quantity INTEGER NOT NULL CHECK (quantity >= 1),
		  PRIMARY KEY (order_id, sku)
		);
		ALTER TABLE order_items
		  ADD COLUMN IF NOT EXISTS unit_price INTEGER;
		ALTER TABLE order_items
		  DROP CONSTRAINT IF EXISTS order_items_unit_price_non_negative;
		ALTER TABLE order_items
		  ADD CONSTRAINT order_items_unit_price_non_negative CHECK (unit_price IS NULL OR unit_price >= 0);
		CREATE TABLE IF NOT EXISTS payments (
		  order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
		  idempotency_key TEXT NOT NULL,
		  status TEXT NOT NULL,
		  amount INTEGER,
		  PRIMARY KEY (order_id, idempotency_key)
		);
		ALTER TABLE payments
		  ADD COLUMN IF NOT EXISTS amount INTEGER;
		ALTER TABLE payments
		  DROP CONSTRAINT IF EXISTS payments_amount_positive_or_null;
		ALTER TABLE payments
		  ADD CONSTRAINT payments_amount_positive_or_null CHECK (amount IS NULL OR amount >= 1);
		CREATE TABLE IF NOT EXISTS product_prices (
		  sku TEXT PRIMARY KEY,
		  unit_price INTEGER NOT NULL CHECK (unit_price >= 0),
		  currency TEXT NOT NULL CHECK (currency = 'JPY')
		);
		CREATE TABLE IF NOT EXISTS inventory (
		  sku TEXT PRIMARY KEY,
		  quantity INTEGER NOT NULL CHECK (quantity >= 0)
		);
		INSERT INTO product_prices (sku, unit_price, currency) VALUES
		  ('sku-1', 100, 'JPY'),
		  ('sku-2', 80, 'JPY'),
		  ('sku-3', 50, 'JPY')
		ON CONFLICT (sku) DO UPDATE
		SET unit_price = EXCLUDED.unit_price, currency = EXCLUDED.currency;
		ALTER TABLE orders
		  ADD COLUMN IF NOT EXISTS customer_id TEXT;
		UPDATE orders SET customer_id = 'unknown' WHERE customer_id IS NULL;
		ALTER TABLE orders
		  ALTER COLUMN customer_id SET NOT NULL;
	`)
	if err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}
}
