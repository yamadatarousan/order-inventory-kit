package domain_test

import (
	"testing"

	"order-inventory-kit/internal/domain"
)

// このテストは Order ドメイン不変条件を固定する。
// 仕様対象: 初期状態 accepted、明細数量制約、同一SKU重複禁止。
// 根拠: 注文生成規則の変更でドメイン性質が崩れないようにするため。
func TestNewOrder_不変条件_初期状態はaccepted(t *testing.T) {
	order, err := domain.NewOrder("order-1", "c-1", []domain.OrderItem{
		{SKU: "sku-1", Quantity: 1},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if order.Status != domain.OrderStatusAccepted {
		t.Fatalf("expected accepted, got %s", order.Status)
	}
	if order.CustomerID != "c-1" {
		t.Fatalf("expected customer c-1, got %s", order.CustomerID)
	}
}

func TestNewOrder_不変条件_数量は1以上(t *testing.T) {
	_, err := domain.NewOrder("order-1", "c-1", []domain.OrderItem{
		{SKU: "sku-1", Quantity: 0},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewOrder_不変条件_同一SKUの重複は禁止(t *testing.T) {
	_, err := domain.NewOrder("order-1", "c-1", []domain.OrderItem{
		{SKU: "sku-1", Quantity: 1},
		{SKU: "sku-1", Quantity: 2},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewOrder_不変条件_IDは必須(t *testing.T) {
	_, err := domain.NewOrder("", "c-1", []domain.OrderItem{
		{SKU: "sku-1", Quantity: 1},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewOrder_不変条件_明細は1件以上必須(t *testing.T) {
	_, err := domain.NewOrder("order-1", "c-1", []domain.OrderItem{})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewOrder_不変条件_SKUは必須(t *testing.T) {
	_, err := domain.NewOrder("order-1", "c-1", []domain.OrderItem{
		{SKU: "", Quantity: 1},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewOrder_不変条件_CustomerIDは必須(t *testing.T) {
	_, err := domain.NewOrder("order-1", "", []domain.OrderItem{
		{SKU: "sku-1", Quantity: 1},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}
