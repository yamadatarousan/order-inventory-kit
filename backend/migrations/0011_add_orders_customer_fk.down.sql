-- orders.customer_id の顧客参照FKを削除する。
ALTER TABLE orders
DROP CONSTRAINT IF EXISTS orders_customer_id_fk;
