-- 開発/テスト用の初期在庫を投入する。
-- 既存SKUがある場合は上書きせず維持する。
INSERT INTO inventory (sku, quantity) VALUES
  ('sku-1', 100),
  ('sku-2', 80),
  ('sku-3', 50)
ON CONFLICT (sku) DO NOTHING;
