package domain_test

import (
	"testing"

	"order-inventory-kit/internal/domain"
)

func TestNewInventory_正常系(t *testing.T) {
	inv, err := domain.NewInventory("sku-1", 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if inv.SKU != "sku-1" {
		t.Fatalf("expected sku-1, got %s", inv.SKU)
	}
	if inv.Quantity != 10 {
		t.Fatalf("expected quantity 10, got %d", inv.Quantity)
	}
}

func TestNewInventory_異常系_数量が負数(t *testing.T) {
	_, err := domain.NewInventory("sku-1", -1)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestInventory_在庫確保で負数にならない(t *testing.T) {
	inv, _ := domain.NewInventory("sku-1", 3)
	if err := inv.Reserve(4); err == nil {
		t.Fatalf("expected error")
	}
	if inv.Quantity != 3 {
		t.Fatalf("expected quantity to remain 3, got %d", inv.Quantity)
	}
}

func TestInventory_在庫戻しで数量が増える(t *testing.T) {
	inv, _ := domain.NewInventory("sku-1", 3)
	if err := inv.Release(2); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if inv.Quantity != 5 {
		t.Fatalf("expected quantity 5, got %d", inv.Quantity)
	}
}
