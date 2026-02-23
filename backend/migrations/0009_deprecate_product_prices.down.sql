-- ロールバック時に旧価格決定元テーブルを復元する。
CREATE TABLE IF NOT EXISTS product_prices (
  sku TEXT PRIMARY KEY,
  unit_price INTEGER NOT NULL CHECK (unit_price >= 0),
  currency TEXT NOT NULL CHECK (currency = 'JPY')
);
COMMENT ON TABLE product_prices IS 'SKUごとのサーバ側価格情報を保持するテーブル';
COMMENT ON COLUMN product_prices.sku IS '商品SKU';
COMMENT ON COLUMN product_prices.unit_price IS '商品単価（最小通貨単位）';
COMMENT ON COLUMN product_prices.currency IS '通貨コード（現状JPY固定）';

-- 復元した product_prices に現行 products の価格情報を戻す。
INSERT INTO product_prices (sku, unit_price, currency)
SELECT sku, unit_price, currency
FROM products
ON CONFLICT (sku) DO UPDATE
SET
  unit_price = EXCLUDED.unit_price,
  currency = EXCLUDED.currency;
