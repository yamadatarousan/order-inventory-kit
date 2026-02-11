package db

import "testing"

type inventoryRepositoryContract interface {
	GetBySKU(sku string) (InventorySnapshot, bool)
	Reserve(sku string, quantity int) (InventorySnapshot, error)
	Release(sku string, quantity int) (InventorySnapshot, error)
}

// InventorySnapshot は契約テストで利用する最小の在庫表現。
type InventorySnapshot struct {
	SKU      string
	Quantity int
}

type inventoryRepositoryAdapter struct {
	repo *InventoryRepository
}

func (a inventoryRepositoryAdapter) GetBySKU(sku string) (InventorySnapshot, bool) {
	inv, ok := a.repo.GetBySKU(sku)
	if !ok {
		return InventorySnapshot{}, false
	}
	return InventorySnapshot{SKU: inv.SKU, Quantity: inv.Quantity}, true
}

func (a inventoryRepositoryAdapter) Reserve(sku string, quantity int) (InventorySnapshot, error) {
	inv, err := a.repo.Reserve(sku, quantity)
	if err != nil {
		return InventorySnapshot{}, err
	}
	return InventorySnapshot{SKU: inv.SKU, Quantity: inv.Quantity}, nil
}

func (a inventoryRepositoryAdapter) Release(sku string, quantity int) (InventorySnapshot, error) {
	inv, err := a.repo.Release(sku, quantity)
	if err != nil {
		return InventorySnapshot{}, err
	}
	return InventorySnapshot{SKU: inv.SKU, Quantity: inv.Quantity}, nil
}

func TestInventoryRepository契約_GetBySKU_存在しないSKUは見つからない(t *testing.T) {
	repo := setupInventoryContractRepo(t)

	inv, ok := repo.GetBySKU("missing-sku")
	if ok {
		t.Fatalf("expected missing-sku to be absent, got %+v", inv)
	}
}

func TestInventoryRepository契約_Reserve_不正数量では失敗し在庫は変化しない(t *testing.T) {
	repo := setupInventoryContractRepo(t)
	seedInventory(t, "sku-1", 10)

	_, err := repo.Reserve("sku-1", 0)
	if err == nil {
		t.Fatalf("expected error for invalid reserve quantity")
	}

	saved, ok := repo.GetBySKU("sku-1")
	if !ok {
		t.Fatalf("expected sku-1 to exist")
	}
	if saved.Quantity != 10 {
		t.Fatalf("expected quantity 10, got %d", saved.Quantity)
	}
}

func TestInventoryRepository契約_Reserve_存在しないSKUでは失敗する(t *testing.T) {
	repo := setupInventoryContractRepo(t)

	_, err := repo.Reserve("missing-sku", 1)
	if err == nil {
		t.Fatalf("expected not found error")
	}
}

func TestInventoryRepository契約_Release_不正数量では失敗し在庫は変化しない(t *testing.T) {
	repo := setupInventoryContractRepo(t)
	seedInventory(t, "sku-1", 10)

	_, err := repo.Release("sku-1", 0)
	if err == nil {
		t.Fatalf("expected error for invalid release quantity")
	}

	saved, ok := repo.GetBySKU("sku-1")
	if !ok {
		t.Fatalf("expected sku-1 to exist")
	}
	if saved.Quantity != 10 {
		t.Fatalf("expected quantity 10, got %d", saved.Quantity)
	}
}

func TestInventoryRepository契約_Release_存在しないSKUでは失敗する(t *testing.T) {
	repo := setupInventoryContractRepo(t)

	_, err := repo.Release("missing-sku", 1)
	if err == nil {
		t.Fatalf("expected not found error")
	}
}

func TestInventoryRepository契約_ReserveとReleaseは永続状態を更新する(t *testing.T) {
	repo := setupInventoryContractRepo(t)
	seedInventory(t, "sku-1", 10)

	if _, err := repo.Reserve("sku-1", 4); err != nil {
		t.Fatalf("reserve failed: %v", err)
	}
	savedAfterReserve, _ := repo.GetBySKU("sku-1")
	if savedAfterReserve.Quantity != 6 {
		t.Fatalf("expected quantity 6 after reserve, got %d", savedAfterReserve.Quantity)
	}

	if _, err := repo.Release("sku-1", 3); err != nil {
		t.Fatalf("release failed: %v", err)
	}
	savedAfterRelease, _ := repo.GetBySKU("sku-1")
	if savedAfterRelease.Quantity != 9 {
		t.Fatalf("expected quantity 9 after release, got %d", savedAfterRelease.Quantity)
	}
}

func setupInventoryContractRepo(t *testing.T) inventoryRepositoryContract {
	t.Helper()

	db := openTestDB(t)
	t.Cleanup(func() {
		_ = db.Close()
	})
	ensureSchema(t, db)
	resetTables(t, db)

	return inventoryRepositoryAdapter{repo: NewInventoryRepository(db)}
}

func seedInventory(t *testing.T, sku string, quantity int) {
	t.Helper()

	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)

	_, err := db.Exec(`INSERT INTO inventory (sku, quantity) VALUES ($1, $2)`, sku, quantity)
	if err != nil {
		t.Fatalf("failed to seed inventory: %v", err)
	}
}
