-- 単価スナップショットの制約を削除する。
ALTER TABLE order_items
DROP CONSTRAINT IF EXISTS order_items_unit_price_non_negative;

-- 注文明細の単価スナップショット列を削除する。
ALTER TABLE order_items
DROP COLUMN IF EXISTS unit_price;

-- 価格情報テーブルを削除する。
DROP TABLE IF EXISTS product_prices;
