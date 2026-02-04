package db

import (
	"testing"

	"order-inventory-kit/internal/domain"
)

func TestOrderRepository_作成と取得(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	repo := NewOrderRepository(db)
	order, _ := domain.NewOrder("order-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 2}})

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
	order, _ := domain.NewOrder("order-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 2}})
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
