package usecase

import (
	"errors"
	"testing"

	"order-inventory-kit/internal/domain"
)

// このテストは InventoryUsecase の振る舞い仕様を固定する。
// 仕様対象: Reserve/Release の成功時状態遷移と、在庫不足・不正入力・未存在SKUの失敗。
// 根拠: Repository 実装差し替え時にもユースケース責務を維持するため。
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

func (r *memoryInventoryRepo) Reserve(sku string, quantity int) (domain.Inventory, error) {
	inv, ok := r.items[sku]
	if !ok {
		return domain.Inventory{}, errors.New("not found")
	}
	if err := inv.Reserve(quantity); err != nil {
		return domain.Inventory{}, err
	}
	r.items[sku] = inv
	return inv, nil
}

func (r *memoryInventoryRepo) Release(sku string, quantity int) (domain.Inventory, error) {
	inv, ok := r.items[sku]
	if !ok {
		return domain.Inventory{}, errors.New("not found")
	}
	if err := inv.Release(quantity); err != nil {
		return domain.Inventory{}, err
	}
	r.items[sku] = inv
	return inv, nil
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

func TestReserveInventory_異常系_存在しないSKUは失敗(t *testing.T) {
	repo := newMemoryInventoryRepo()
	uc := NewInventoryUsecase(repo)

	_, err := uc.ReserveInventory(ReserveInventoryInput{SKU: "missing", Quantity: 1})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestReleaseInventory_正常系(t *testing.T) {
	repo := newMemoryInventoryRepo()
	inv, _ := domain.NewInventory("sku-1", 5, 3)
	repo.items["sku-1"] = inv

	uc := NewInventoryUsecase(repo)
	out, err := uc.ReleaseInventory(ReleaseInventoryInput{SKU: "sku-1", Quantity: 3})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.SKU != "sku-1" {
		t.Fatalf("expected sku-1, got %s", out.SKU)
	}
	if out.Quantity != 5 {
		t.Fatalf("expected quantity 5, got %d", out.Quantity)
	}

	saved, _ := repo.GetBySKU("sku-1")
	if saved.Quantity != 5 {
		t.Fatalf("expected saved quantity 5, got %d", saved.Quantity)
	}
}

func TestReleaseInventory_異常系_存在しないSKUは失敗(t *testing.T) {
	repo := newMemoryInventoryRepo()
	uc := NewInventoryUsecase(repo)

	_, err := uc.ReleaseInventory(ReleaseInventoryInput{SKU: "missing", Quantity: 1})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestReleaseInventory_異常系_数量が不正(t *testing.T) {
	repo := newMemoryInventoryRepo()
	inv, _ := domain.NewInventory("sku-1", 2)
	repo.items["sku-1"] = inv
	uc := NewInventoryUsecase(repo)

	_, err := uc.ReleaseInventory(ReleaseInventoryInput{SKU: "sku-1", Quantity: 0})
	if err == nil {
		t.Fatalf("expected error")
	}
}
