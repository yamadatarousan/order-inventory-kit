-- 旧モデルへ戻すため、quantityを on_hand-reserved で復元する。
UPDATE inventory
SET quantity = on_hand - reserved;

-- 追加した制約を削除する。
ALTER TABLE inventory
DROP CONSTRAINT IF EXISTS inventory_reserved_non_negative,
DROP CONSTRAINT IF EXISTS inventory_on_hand_non_negative;

-- 追加した列を削除する。
ALTER TABLE inventory
DROP COLUMN IF EXISTS reserved,
DROP COLUMN IF EXISTS on_hand;
