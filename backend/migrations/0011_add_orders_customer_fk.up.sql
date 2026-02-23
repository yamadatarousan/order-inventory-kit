-- 外部参照が解決できない注文のcustomer_idを暫定値へ寄せる。
UPDATE orders
SET customer_id = 'unknown'
WHERE NOT EXISTS (
  SELECT 1 FROM customers WHERE customers.id = orders.customer_id
);

-- orders.customer_id の顧客参照整合をFKで強制する。
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'orders_customer_id_fk'
  ) THEN
    ALTER TABLE orders
      ADD CONSTRAINT orders_customer_id_fk
      FOREIGN KEY (customer_id)
      REFERENCES customers(id);
  END IF;
END $$;
