package usecase

import (
	"testing"

	"order-inventory-kit/internal/domain"
)

type memoryInventoryRepo struct {
	items map[string]domain.Inventory
}

func newMemoryInventoryRepo() *memoryInventoryRepo {
	return &memoryInventoryRepo{items: make(map[string]domain.Inventory)}
}

func (r *memoryInventoryRepo) GetBySKU(sku string) (domain.Inventory, bool) {
	inv, ok := r.items[sku]
	return inv, ok
}

func (r *memoryInventoryRepo) Update(inventory domain.Inventory) error {
	r.items[inventory.SKU] = inventory
	return nil
}

func TestReserveInventory_正常系(t *testing.T) {
	repo := newMemoryInventoryRepo()
	inv, _ := domain.NewInventory("sku-1", 5)
	repo.items["sku-1"] = inv

	uc := NewInventoryUsecase(repo)
	out, err := uc.ReserveInventory(ReserveInventoryInput{SKU: "sku-1", Quantity: 2})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.SKU != "sku-1" {
		t.Fatalf("expected sku-1, got %s", out.SKU)
	}
	if out.RemainingQuantity != 3 {
		t.Fatalf("expected remaining 3, got %d", out.RemainingQuantity)
	}

	saved, _ := repo.GetBySKU("sku-1")
	if saved.Quantity != 3 {
		t.Fatalf("expected saved quantity 3, got %d", saved.Quantity)
	}
}

func TestReserveInventory_異常系_在庫不足は失敗(t *testing.T) {
	repo := newMemoryInventoryRepo()
	inv, _ := domain.NewInventory("sku-1", 1)
	repo.items["sku-1"] = inv

	uc := NewInventoryUsecase(repo)
	_, err := uc.ReserveInventory(ReserveInventoryInput{SKU: "sku-1", Quantity: 2})
	if err == nil {
		t.Fatalf("expected error")
	}

	saved, _ := repo.GetBySKU("sku-1")
	if saved.Quantity != 1 {
		t.Fatalf("expected quantity to remain 1, got %d", saved.Quantity)
	}
}

func TestReserveInventory_異常系_存在しないSKUは失敗(t *testing.T) {
	repo := newMemoryInventoryRepo()
	uc := NewInventoryUsecase(repo)

	_, err := uc.ReserveInventory(ReserveInventoryInput{SKU: "missing", Quantity: 1})
	if err == nil {
		t.Fatalf("expected error")
	}
}
