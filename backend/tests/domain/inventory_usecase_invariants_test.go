package domain_test

import (
	"errors"
	"testing"

	"order-inventory-kit/internal/domain"
	"order-inventory-kit/internal/usecase"
)

// このテストは InventoryUsecase 経由で観測される不変条件を固定する。
// 仕様対象: 在庫不足時に確保が失敗し、在庫数量が変化しないこと。
// 根拠: ユースケース実装を変更しても在庫下限制約が破れないようにするため。
type 在庫不変条件用Repo struct {
	items map[string]domain.Inventory
}

func new在庫不変条件用Repo() *在庫不変条件用Repo {
	return &在庫不変条件用Repo{items: make(map[string]domain.Inventory)}
}

func (r *在庫不変条件用Repo) GetBySKU(sku string) (domain.Inventory, bool) {
	inv, ok := r.items[sku]
	return inv, ok
}

func (r *在庫不変条件用Repo) Reserve(sku string, quantity int) (domain.Inventory, error) {
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

func (r *在庫不変条件用Repo) Release(sku string, quantity int) (domain.Inventory, error) {
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

func TestReserveInventory_不変条件_在庫不足では数量が減らない(t *testing.T) {
	repo := new在庫不変条件用Repo()
	inv, _ := domain.NewInventory("sku-1", 1)
	repo.items["sku-1"] = inv

	uc := usecase.NewInventoryUsecase(repo)
	_, err := uc.ReserveInventory(usecase.ReserveInventoryInput{SKU: "sku-1", Quantity: 2})
	if err == nil {
		t.Fatalf("expected error")
	}

	saved, _ := repo.GetBySKU("sku-1")
	if saved.Quantity != 1 {
		t.Fatalf("expected quantity to remain 1, got %d", saved.Quantity)
	}
}
