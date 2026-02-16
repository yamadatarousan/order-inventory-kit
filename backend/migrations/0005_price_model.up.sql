-- 価格情報の決定元テーブルを追加する。
CREATE TABLE IF NOT EXISTS product_prices (
  sku TEXT PRIMARY KEY,
  unit_price INTEGER NOT NULL CHECK (unit_price >= 0),
  currency TEXT NOT NULL CHECK (currency = 'JPY')
);
COMMENT ON TABLE product_prices IS 'SKUごとのサーバ側価格情報を保持するテーブル';
COMMENT ON COLUMN product_prices.sku IS '商品SKU';
COMMENT ON COLUMN product_prices.unit_price IS '商品単価（最小通貨単位）';
COMMENT ON COLUMN product_prices.currency IS '通貨コード（現状JPY固定）';

-- 注文明細に単価スナップショット列を追加する。
ALTER TABLE order_items
ADD COLUMN IF NOT EXISTS unit_price INTEGER;
COMMENT ON COLUMN order_items.unit_price IS '注文作成時点の単価スナップショット（最小通貨単位）';

-- 単価が設定される場合は負数を許可しない。
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'order_items_unit_price_non_negative'
  ) THEN
    ALTER TABLE order_items
      ADD CONSTRAINT order_items_unit_price_non_negative CHECK (unit_price IS NULL OR unit_price >= 0);
  END IF;
END $$;

-- 開発/テストで使う初期価格情報を投入する。
INSERT INTO product_prices (sku, unit_price, currency) VALUES
  ('sku-1', 100, 'JPY'),
  ('sku-2', 80, 'JPY'),
  ('sku-3', 50, 'JPY')
ON CONFLICT (sku) DO UPDATE
SET
  unit_price = EXCLUDED.unit_price,
  currency = EXCLUDED.currency;
