-- 初期スキーマを作成する（注文・注文明細・決済・在庫）。
CREATE TABLE IF NOT EXISTS orders (
  id TEXT PRIMARY KEY,
  status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
COMMENT ON TABLE orders IS '注文のヘッダ情報を保持するテーブル';
COMMENT ON COLUMN orders.id IS '注文ID';
COMMENT ON COLUMN orders.status IS '注文ステータス（accepted/confirmed/canceled）';
COMMENT ON COLUMN orders.created_at IS '注文作成日時';

-- 注文ごとのSKUと数量を保持する。
CREATE TABLE IF NOT EXISTS order_items (
  order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  sku TEXT NOT NULL,
  quantity INTEGER NOT NULL CHECK (quantity >= 1),
  PRIMARY KEY (order_id, sku)
);
COMMENT ON TABLE order_items IS '注文に含まれる商品明細を保持するテーブル';
COMMENT ON COLUMN order_items.order_id IS '注文明細が属する注文ID';
COMMENT ON COLUMN order_items.sku IS '商品SKU';
COMMENT ON COLUMN order_items.quantity IS '注文数量（1以上）';

-- 注文ごとの冪等キー付き決済記録を保持する。
CREATE TABLE IF NOT EXISTS payments (
  order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  idempotency_key TEXT NOT NULL,
  status TEXT NOT NULL,
  PRIMARY KEY (order_id, idempotency_key)
);
COMMENT ON TABLE payments IS '決済確定の冪等記録を保持するテーブル';
COMMENT ON COLUMN payments.order_id IS '決済対象の注文ID';
COMMENT ON COLUMN payments.idempotency_key IS '冪等性キー';
COMMENT ON COLUMN payments.status IS '決済ステータス';

-- 在庫をSKU単位で保持する（旧モデル: quantity）。
CREATE TABLE IF NOT EXISTS inventory (
  sku TEXT PRIMARY KEY,
  quantity INTEGER NOT NULL CHECK (quantity >= 0)
);
COMMENT ON TABLE inventory IS '在庫情報を保持するテーブル';
COMMENT ON COLUMN inventory.sku IS '商品SKU';
COMMENT ON COLUMN inventory.quantity IS '旧在庫モデルの在庫数量（0以上）';
