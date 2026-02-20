-- 在庫引当の注文明細単位追跡テーブルを追加する。
CREATE TABLE IF NOT EXISTS inventory_reservations (
  order_id TEXT NOT NULL,
  sku TEXT NOT NULL,
  quantity INTEGER NOT NULL CHECK (quantity >= 1),
  PRIMARY KEY (order_id, sku),
  CONSTRAINT inventory_reservations_order_item_fk
    FOREIGN KEY (order_id, sku)
    REFERENCES order_items(order_id, sku)
    ON DELETE CASCADE
);
COMMENT ON TABLE inventory_reservations IS '注文ごとの在庫引当数量を明細単位で保持するテーブル';
COMMENT ON COLUMN inventory_reservations.order_id IS '引当対象の注文ID';
COMMENT ON COLUMN inventory_reservations.sku IS '引当対象の商品SKU';
COMMENT ON COLUMN inventory_reservations.quantity IS '引当数量（1以上）';

-- 引当台帳は注文操作から派生して作成されるため、初期seedは投入しない。
