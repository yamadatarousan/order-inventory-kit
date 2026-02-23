-- 顧客マスタを追加し、注文時の顧客参照判定に使う。
CREATE TABLE IF NOT EXISTS customers (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE
);
COMMENT ON TABLE customers IS '顧客マスタ（注文時の顧客参照整合に利用）';
COMMENT ON COLUMN customers.id IS '顧客ID';
COMMENT ON COLUMN customers.name IS '顧客名';
COMMENT ON COLUMN customers.is_active IS '有効フラグ（TRUE: 有効, FALSE: 無効）';

-- 開発/テストで使う初期顧客を投入する。
INSERT INTO customers (id, name, is_active) VALUES
  ('c-1', '顧客c-1', TRUE),
  ('c-2', '顧客c-2', TRUE),
  ('unknown', '未設定顧客', FALSE)
ON CONFLICT (id) DO UPDATE
SET
  name = EXCLUDED.name,
  is_active = EXCLUDED.is_active;
