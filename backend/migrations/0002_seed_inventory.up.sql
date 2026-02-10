INSERT INTO inventory (sku, quantity) VALUES
  ('sku-1', 100),
  ('sku-2', 80),
  ('sku-3', 50)
ON CONFLICT (sku) DO NOTHING;
