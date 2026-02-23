-- 商品マスタを追加し、価格決定元を products に統合する。
CREATE TABLE IF NOT EXISTS products (
  sku TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  unit_price INTEGER NOT NULL CHECK (unit_price >= 0),
  currency TEXT NOT NULL CHECK (currency = 'JPY'),
  is_active BOOLEAN NOT NULL DEFAULT TRUE
);
COMMENT ON TABLE products IS '商品マスタ（販売状態と価格の決定元）';
COMMENT ON COLUMN products.sku IS '商品SKU';
COMMENT ON COLUMN products.name IS '商品名';
COMMENT ON COLUMN products.unit_price IS '商品単価（最小通貨単位）';
COMMENT ON COLUMN products.currency IS '通貨コード（現状JPY固定）';
COMMENT ON COLUMN products.is_active IS '販売中フラグ（TRUE: 販売中, FALSE: 販売停止）';

-- 既存価格テーブルから商品マスタへ初期データを投入する。
INSERT INTO products (sku, name, unit_price, currency, is_active)
SELECT
  product_prices.sku,
  product_prices.sku,
  product_prices.unit_price,
  product_prices.currency,
  TRUE
FROM product_prices
ON CONFLICT (sku) DO UPDATE
SET
  unit_price = EXCLUDED.unit_price,
  currency = EXCLUDED.currency;
