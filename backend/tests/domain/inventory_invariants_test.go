package domain_test

import (
	"testing"

	"order-inventory-kit/internal/domain"
)

// このテストは Inventory ドメイン不変条件を固定する。
// 仕様対象: 在庫数量の非負制約、確保時の下限維持、戻し時の数量増加。
// 根拠: ドメイン実装の変更で在庫の基本性質が崩れないようにするため。
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
