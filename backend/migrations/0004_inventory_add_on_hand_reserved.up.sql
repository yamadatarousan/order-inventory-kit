-- 在庫標準モデルへの移行として on_hand / reserved 列を追加する。
ALTER TABLE inventory
ADD COLUMN IF NOT EXISTS on_hand INTEGER,
ADD COLUMN IF NOT EXISTS reserved INTEGER;
COMMENT ON TABLE inventory IS '在庫情報を保持するテーブル（標準モデル: on_hand/reserved/available）';
COMMENT ON COLUMN inventory.on_hand IS '実在庫（0以上）';
COMMENT ON COLUMN inventory.reserved IS '引当済み在庫（0以上）';

-- 既存quantityから初期値をバックフィルする（on_hand=quantity, reserved=0）。
UPDATE inventory
SET
  on_hand = COALESCE(on_hand, quantity),
  reserved = COALESCE(reserved, 0);

-- 新列を必須 + 既定値付きにする。
ALTER TABLE inventory
ALTER COLUMN on_hand SET NOT NULL,
ALTER COLUMN reserved SET NOT NULL,
ALTER COLUMN on_hand SET DEFAULT 0,
ALTER COLUMN reserved SET DEFAULT 0;

-- on_handの非負制約を追加する（重複追加は避ける）。
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'inventory_on_hand_non_negative'
  ) THEN
    ALTER TABLE inventory
      ADD CONSTRAINT inventory_on_hand_non_negative CHECK (on_hand >= 0);
  END IF;
END $$;

-- reservedの非負制約を追加する（重複追加は避ける）。
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'inventory_reserved_non_negative'
  ) THEN
    ALTER TABLE inventory
      ADD CONSTRAINT inventory_reserved_non_negative CHECK (reserved >= 0);
  END IF;
END $$;
