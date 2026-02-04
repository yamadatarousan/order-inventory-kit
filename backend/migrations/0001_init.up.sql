CREATE TABLE IF NOT EXISTS orders (
  id TEXT PRIMARY KEY,
  status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS order_items (
  order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  sku TEXT NOT NULL,
  quantity INTEGER NOT NULL CHECK (quantity >= 1),
  PRIMARY KEY (order_id, sku)
);

CREATE TABLE IF NOT EXISTS payments (
  order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  idempotency_key TEXT NOT NULL,
  status TEXT NOT NULL,
  PRIMARY KEY (order_id, idempotency_key)
);

CREATE TABLE IF NOT EXISTS inventory (
  sku TEXT PRIMARY KEY,
  quantity INTEGER NOT NULL CHECK (quantity >= 0)
);
