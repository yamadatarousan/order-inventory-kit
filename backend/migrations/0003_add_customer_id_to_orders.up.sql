-- ordersにcustomer_id列を追加する。
ALTER TABLE orders
ADD COLUMN IF NOT EXISTS customer_id TEXT;
COMMENT ON COLUMN orders.customer_id IS '注文者の顧客ID';

-- 既存レコードのNULLを暫定値で埋める。
UPDATE orders
SET customer_id = 'unknown'
WHERE customer_id IS NULL;

-- customer_idを必須化する。
ALTER TABLE orders
ALTER COLUMN customer_id SET NOT NULL;
