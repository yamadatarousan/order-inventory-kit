ALTER TABLE orders
ADD COLUMN IF NOT EXISTS customer_id TEXT;

UPDATE orders
SET customer_id = 'unknown'
WHERE customer_id IS NULL;

ALTER TABLE orders
ALTER COLUMN customer_id SET NOT NULL;
