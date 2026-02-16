-- amount制約を削除する。
ALTER TABLE payments
DROP CONSTRAINT IF EXISTS payments_amount_positive_or_null;

-- 決済要求額列を削除する。
ALTER TABLE payments
DROP COLUMN IF EXISTS amount;
