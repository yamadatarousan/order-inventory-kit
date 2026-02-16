-- 開発/テスト用の初期在庫を投入する。
-- 標準在庫モデル（on_hand/reserved）が存在する環境ではそれを優先し、
-- 旧モデル（quantityのみ）の環境では従来形式で投入する。
-- 既存SKUがある場合は上書きせず維持する。
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'inventory' AND column_name = 'on_hand'
  ) AND EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'inventory' AND column_name = 'reserved'
  ) THEN
    INSERT INTO inventory (sku, quantity, on_hand, reserved) VALUES
      ('sku-1', 100, 100, 0),
      ('sku-2', 80, 80, 0),
      ('sku-3', 50, 50, 0)
    ON CONFLICT (sku) DO NOTHING;
  ELSE
    INSERT INTO inventory (sku, quantity) VALUES
      ('sku-1', 100),
      ('sku-2', 80),
      ('sku-3', 50)
    ON CONFLICT (sku) DO NOTHING;
  END IF;
END $$;
