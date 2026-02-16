-- 決済記録に決済要求額（最小通貨単位）を保存する列を追加する。
ALTER TABLE payments
ADD COLUMN IF NOT EXISTS amount INTEGER;
COMMENT ON COLUMN payments.amount IS '決済要求額（最小通貨単位）。旧データは未保存のためNULLを許容する';

-- amountの入力制約を追加する（NULLまたは1以上）。
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'payments_amount_positive_or_null'
  ) THEN
    ALTER TABLE payments
      ADD CONSTRAINT payments_amount_positive_or_null CHECK (amount IS NULL OR amount >= 1);
  END IF;
END $$;

-- 既存データ移行方針:
-- 過去決済は amount を保持していないため復元不能。既存行は NULL のまま保持し、
-- このマイグレーション適用後に作成される決済から amount を保存する。
