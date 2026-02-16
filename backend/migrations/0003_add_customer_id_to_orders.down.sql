-- customer_id列を削除する。
ALTER TABLE orders
DROP COLUMN IF EXISTS customer_id;
