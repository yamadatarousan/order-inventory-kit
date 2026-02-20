package db

import "testing"

// このテストは InventoryRepository の最小契約を固定する。
// 契約対象: GetBySKU / Reserve / Release の戻り値・エラー・永続状態（on_hand/reserved/available）。
// 根拠: UseCase が Repository 実装を差し替えても同じ振る舞いを前提にできるようにするため。
// 失敗時に在庫数量が変化しないことを含め、回帰をCIで検出する。
type inventoryRepositoryContract interface {
	GetBySKU(sku string) (InventorySnapshot, bool)
	Reserve(sku string, quantity int) (InventorySnapshot, error)
	Release(sku string, quantity int) (InventorySnapshot, error)
}

// InventorySnapshot は契約テストで利用する最小の在庫表現。
type InventorySnapshot struct {
	SKU       string
	OnHand    int
	Reserved  int
	Available int
}

type inventoryRepositoryAdapter struct {
	repo *InventoryRepository
}

func (a inventoryRepositoryAdapter) GetBySKU(sku string) (InventorySnapshot, bool) {
	inv, ok := a.repo.GetBySKU(sku)
	if !ok {
		return InventorySnapshot{}, false
	}
	return InventorySnapshot{
		SKU:       inv.SKU,
		OnHand:    inv.OnHand,
		Reserved:  inv.Reserved,
		Available: inv.Available(),
	}, true
}

func (a inventoryRepositoryAdapter) Reserve(sku string, quantity int) (InventorySnapshot, error) {
	inv, err := a.repo.Reserve(sku, quantity)
	if err != nil {
		return InventorySnapshot{}, err
	}
	return InventorySnapshot{
		SKU:       inv.SKU,
		OnHand:    inv.OnHand,
		Reserved:  inv.Reserved,
		Available: inv.Available(),
	}, nil
}

func (a inventoryRepositoryAdapter) Release(sku string, quantity int) (InventorySnapshot, error) {
	inv, err := a.repo.Release(sku, quantity)
	if err != nil {
		return InventorySnapshot{}, err
	}
	return InventorySnapshot{
		SKU:       inv.SKU,
		OnHand:    inv.OnHand,
		Reserved:  inv.Reserved,
		Available: inv.Available(),
	}, nil
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
	seedInventory(t, "sku-1", 10, 0)

	_, err := repo.Reserve("sku-1", 0)
	if err == nil {
		t.Fatalf("expected error for invalid reserve quantity")
	}

	saved, ok := repo.GetBySKU("sku-1")
	if !ok {
		t.Fatalf("expected sku-1 to exist")
	}
	if saved.OnHand != 10 || saved.Reserved != 0 || saved.Available != 10 {
		t.Fatalf("expected (on_hand,reserved,available)=(10,0,10), got (%d,%d,%d)", saved.OnHand, saved.Reserved, saved.Available)
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
	seedInventory(t, "sku-1", 10, 0)

	_, err := repo.Release("sku-1", 0)
	if err == nil {
		t.Fatalf("expected error for invalid release quantity")
	}

	saved, ok := repo.GetBySKU("sku-1")
	if !ok {
		t.Fatalf("expected sku-1 to exist")
	}
	if saved.OnHand != 10 || saved.Reserved != 0 || saved.Available != 10 {
		t.Fatalf("expected (on_hand,reserved,available)=(10,0,10), got (%d,%d,%d)", saved.OnHand, saved.Reserved, saved.Available)
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
	seedInventory(t, "sku-1", 10, 0)

	if _, err := repo.Reserve("sku-1", 4); err != nil {
		t.Fatalf("reserve failed: %v", err)
	}
	savedAfterReserve, _ := repo.GetBySKU("sku-1")
	if savedAfterReserve.OnHand != 10 || savedAfterReserve.Reserved != 4 || savedAfterReserve.Available != 6 {
		t.Fatalf("expected after reserve (10,4,6), got (%d,%d,%d)", savedAfterReserve.OnHand, savedAfterReserve.Reserved, savedAfterReserve.Available)
	}

	if _, err := repo.Release("sku-1", 3); err != nil {
		t.Fatalf("release failed: %v", err)
	}
	savedAfterRelease, _ := repo.GetBySKU("sku-1")
	if savedAfterRelease.OnHand != 10 || savedAfterRelease.Reserved != 1 || savedAfterRelease.Available != 9 {
		t.Fatalf("expected after release (10,1,9), got (%d,%d,%d)", savedAfterRelease.OnHand, savedAfterRelease.Reserved, savedAfterRelease.Available)
	}
}

func TestInventoryRepository契約_Release_確保超過の戻しは失敗し永続状態が変化しない(t *testing.T) {
	repo := setupInventoryContractRepo(t)
	seedInventory(t, "sku-1", 10, 2)

	_, err := repo.Release("sku-1", 3)
	if err == nil {
		t.Fatalf("expected error for release over reserved")
	}

	saved, ok := repo.GetBySKU("sku-1")
	if !ok {
		t.Fatalf("expected sku-1 to exist")
	}
	if saved.OnHand != 10 || saved.Reserved != 2 || saved.Available != 8 {
		t.Fatalf("expected unchanged (10,2,8), got (%d,%d,%d)", saved.OnHand, saved.Reserved, saved.Available)
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

func seedInventory(t *testing.T, sku string, onHand int, reserved int) {
	t.Helper()

	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)

	_, err := db.Exec(`
		INSERT INTO inventory (sku, quantity, on_hand, reserved) VALUES ($1, $2, $3, $4)
	`, sku, onHand-reserved, onHand, reserved)
	if err != nil {
		t.Fatalf("failed to seed inventory: %v", err)
	}
}
