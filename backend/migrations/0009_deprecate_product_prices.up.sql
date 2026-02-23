-- product_prices の残存データを products に統合する。
DO $$
BEGIN
  IF to_regclass('public.product_prices') IS NOT NULL THEN
    INSERT INTO products (sku, name, unit_price, currency, is_active)
    SELECT
      product_prices.sku,
      product_prices.sku,
      product_prices.unit_price,
      product_prices.currency,
      TRUE
    FROM product_prices
    ON CONFLICT (sku) DO UPDATE
    SET
      unit_price = EXCLUDED.unit_price,
      currency = EXCLUDED.currency;
  END IF;
END $$;

-- 旧価格決定元テーブルを廃止する。
DROP TABLE IF EXISTS product_prices;
