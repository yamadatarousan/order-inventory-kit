package domain_test

import (
	"testing"

	"order-inventory-kit/internal/domain"
)

func TestNewOrder_不変条件_初期状態はaccepted(t *testing.T) {
	order, err := domain.NewOrder("order-1", []domain.OrderItem{
		{SKU: "sku-1", Quantity: 1},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if order.Status != domain.OrderStatusAccepted {
		t.Fatalf("expected accepted, got %s", order.Status)
	}
}

func TestNewOrder_不変条件_数量は1以上(t *testing.T) {
	_, err := domain.NewOrder("order-1", []domain.OrderItem{
		{SKU: "sku-1", Quantity: 0},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewOrder_不変条件_同一SKUの重複は禁止(t *testing.T) {
	_, err := domain.NewOrder("order-1", []domain.OrderItem{
		{SKU: "sku-1", Quantity: 1},
		{SKU: "sku-1", Quantity: 2},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}
