package db

import (
	"database/sql"
	"testing"

	"order-inventory-kit/internal/domain"
)

// このテストは OrderRepository の永続化仕様を固定する。
// 仕様対象: 注文の作成・取得・更新で Order の状態と明細が正しく保存されること。
// 根拠: 保存方式を変更しても注文状態遷移の基盤となるデータ整合を維持するため。
func TestOrderRepository_作成と取得(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	repo := NewOrderRepository(db)
	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 2}})

	if err := repo.Create(order); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	got, ok := repo.Get("order-1")
	if !ok {
		t.Fatalf("expected order to exist")
	}
	if got.Status != domain.OrderStatusAccepted {
		t.Fatalf("expected accepted, got %s", got.Status)
	}
	if got.CustomerID != "c-1" {
		t.Fatalf("expected customer id c-1, got %s", got.CustomerID)
	}
	if len(got.Items) != 1 || got.Items[0].SKU != "sku-1" {
		t.Fatalf("expected items to be saved")
	}
}

func TestOrderRepository_更新(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	repo := NewOrderRepository(db)
	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 2}})
	_ = repo.Create(order)

	order.Status = domain.OrderStatusCanceled
	if err := repo.Update(order); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	got, _ := repo.Get("order-1")
	if got.Status != domain.OrderStatusCanceled {
		t.Fatalf("expected canceled, got %s", got.Status)
	}
}

func TestOrderRepository_作成時に単価スナップショットを保存する(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	repo := NewOrderRepository(db)
	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 2}})

	if err := repo.Create(order); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var savedUnitPrice sql.NullInt64
	if err := db.QueryRow(`SELECT unit_price FROM order_items WHERE order_id = $1 AND sku = $2`, "order-1", "sku-1").Scan(&savedUnitPrice); err != nil {
		t.Fatalf("failed to load unit_price: %v", err)
	}
	if !savedUnitPrice.Valid {
		t.Fatalf("expected unit_price to be set")
	}
	if savedUnitPrice.Int64 != 100 {
		t.Fatalf("expected unit_price 100, got %d", savedUnitPrice.Int64)
	}

	if _, err := db.Exec(`UPDATE product_prices SET unit_price = 999 WHERE sku = $1`, "sku-1"); err != nil {
		t.Fatalf("failed to update product price: %v", err)
	}

	var snapshotPrice int
	if err := db.QueryRow(`SELECT unit_price FROM order_items WHERE order_id = $1 AND sku = $2`, "order-1", "sku-1").Scan(&snapshotPrice); err != nil {
		t.Fatalf("failed to reload snapshot unit_price: %v", err)
	}
	if snapshotPrice != 100 {
		t.Fatalf("snapshot unit_price must remain 100, got %d", snapshotPrice)
	}
}

func TestOrderRepository_価格情報が存在しないSKUを含む注文は失敗する(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	repo := NewOrderRepository(db)
	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "missing-sku", Quantity: 1}})

	if err := repo.Create(order); err == nil {
		t.Fatalf("expected create to fail when price is missing")
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM orders WHERE id = $1`, "order-1").Scan(&count); err != nil {
		t.Fatalf("failed to count order rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("order row must not be persisted on failure, got %d", count)
	}
}
