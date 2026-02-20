package db

import "testing"

// このテストは InventoryRepository のDB実装が基本仕様を満たすことを固定する。
// 仕様対象: GetBySKU / Reserve / Release の正常系と在庫不足時の失敗時挙動。
// 根拠: 在庫更新実装の回収や最適化を行っても、UseCase 観点の基本動作を維持するため。
func TestInventoryRepository_GetBySKU_存在する場合(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	_, err := db.Exec(`INSERT INTO inventory (sku, quantity, on_hand, reserved) VALUES ($1, $2, $3, $4)`, "sku-1", 8, 10, 2)
	if err != nil {
		t.Fatalf("failed to insert inventory: %v", err)
	}

	repo := NewInventoryRepository(db)
	inv, ok := repo.GetBySKU("sku-1")
	if !ok {
		t.Fatalf("expected inventory to exist")
	}
	if inv.SKU != "sku-1" {
		t.Fatalf("expected sku-1, got %s", inv.SKU)
	}
	if inv.OnHand != 10 || inv.Reserved != 2 || inv.Available() != 8 {
		t.Fatalf("expected (on_hand,reserved,available)=(10,2,8), got (%d,%d,%d)", inv.OnHand, inv.Reserved, inv.Available())
	}
}

func TestInventoryRepository_Reserve_正常系(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	_, err := db.Exec(`INSERT INTO inventory (sku, quantity, on_hand, reserved) VALUES ($1, $2, $3, $4)`, "sku-1", 5, 5, 0)
	if err != nil {
		t.Fatalf("failed to insert inventory: %v", err)
	}

	repo := NewInventoryRepository(db)
	inv, err := repo.Reserve("sku-1", 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if inv.OnHand != 5 || inv.Reserved != 2 || inv.Available() != 3 {
		t.Fatalf("expected (on_hand,reserved,available)=(5,2,3), got (%d,%d,%d)", inv.OnHand, inv.Reserved, inv.Available())
	}
}

func TestInventoryRepository_Reserve_異常系_在庫不足(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	_, err := db.Exec(`INSERT INTO inventory (sku, quantity, on_hand, reserved) VALUES ($1, $2, $3, $4)`, "sku-1", 1, 1, 0)
	if err != nil {
		t.Fatalf("failed to insert inventory: %v", err)
	}

	repo := NewInventoryRepository(db)
	_, err = repo.Reserve("sku-1", 2)
	if err == nil {
		t.Fatalf("expected error")
	}

	inv, _ := repo.GetBySKU("sku-1")
	if inv.OnHand != 1 || inv.Reserved != 0 || inv.Available() != 1 {
		t.Fatalf("expected unchanged (on_hand,reserved,available)=(1,0,1), got (%d,%d,%d)", inv.OnHand, inv.Reserved, inv.Available())
	}
}

func TestInventoryRepository_Release_正常系(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	_, err := db.Exec(`INSERT INTO inventory (sku, quantity, on_hand, reserved) VALUES ($1, $2, $3, $4)`, "sku-1", 7, 10, 3)
	if err != nil {
		t.Fatalf("failed to insert inventory: %v", err)
	}

	repo := NewInventoryRepository(db)
	inv, err := repo.Release("sku-1", 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if inv.OnHand != 10 || inv.Reserved != 1 || inv.Available() != 9 {
		t.Fatalf("expected (on_hand,reserved,available)=(10,1,9), got (%d,%d,%d)", inv.OnHand, inv.Reserved, inv.Available())
	}
}
