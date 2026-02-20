package db

import (
	"sync"
	"testing"
	"time"
)

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

func TestInventoryRepository_同時実行_Reserve競合でも売り越さない(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	_, err := db.Exec(`INSERT INTO inventory (sku, quantity, on_hand, reserved) VALUES ($1, $2, $3, $4)`, "sku-1", 10, 10, 0)
	if err != nil {
		t.Fatalf("failed to insert inventory: %v", err)
	}

	repo := NewInventoryRepository(db)
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, reserveErr := repo.Reserve("sku-1", 7)
			results <- reserveErr
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	success := 0
	failed := 0
	for reserveErr := range results {
		if reserveErr == nil {
			success++
			continue
		}
		failed++
	}
	if success != 1 || failed != 1 {
		t.Fatalf("expected one success and one failure, got success=%d failed=%d", success, failed)
	}

	inv, ok := repo.GetBySKU("sku-1")
	if !ok {
		t.Fatalf("expected sku-1 to exist")
	}
	if inv.OnHand != 10 || inv.Reserved != 7 || inv.Available() != 3 {
		t.Fatalf("expected final (on_hand,reserved,available)=(10,7,3), got (%d,%d,%d)", inv.OnHand, inv.Reserved, inv.Available())
	}
}

func TestInventoryRepository_同時実行_Release競合でもReservedは負数にならない(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	_, err := db.Exec(`INSERT INTO inventory (sku, quantity, on_hand, reserved) VALUES ($1, $2, $3, $4)`, "sku-1", 7, 10, 3)
	if err != nil {
		t.Fatalf("failed to insert inventory: %v", err)
	}

	repo := NewInventoryRepository(db)
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, releaseErr := repo.Release("sku-1", 2)
			results <- releaseErr
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	success := 0
	failed := 0
	for releaseErr := range results {
		if releaseErr == nil {
			success++
			continue
		}
		failed++
	}
	if success != 1 || failed != 1 {
		t.Fatalf("expected one success and one failure, got success=%d failed=%d", success, failed)
	}

	inv, ok := repo.GetBySKU("sku-1")
	if !ok {
		t.Fatalf("expected sku-1 to exist")
	}
	if inv.OnHand != 10 || inv.Reserved != 1 || inv.Available() != 9 {
		t.Fatalf("expected final (on_hand,reserved,available)=(10,1,9), got (%d,%d,%d)", inv.OnHand, inv.Reserved, inv.Available())
	}
}

func TestInventoryRepository_行ロック_ReserveはFOR_UPDATE解除まで待機する(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	_, err := db.Exec(`INSERT INTO inventory (sku, quantity, on_hand, reserved) VALUES ($1, $2, $3, $4)`, "sku-1", 10, 10, 0)
	if err != nil {
		t.Fatalf("failed to insert inventory: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	var lockedSKU string
	if err := tx.QueryRow(`SELECT sku FROM inventory WHERE sku = $1 FOR UPDATE`, "sku-1").Scan(&lockedSKU); err != nil {
		t.Fatalf("failed to lock row: %v", err)
	}

	repo := NewInventoryRepository(db)
	done := make(chan error, 1)
	go func() {
		_, reserveErr := repo.Reserve("sku-1", 1)
		done <- reserveErr
	}()

	select {
	case err := <-done:
		t.Fatalf("reserve must wait for row lock release, got early result: %v", err)
	case <-time.After(150 * time.Millisecond):
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("failed to commit lock tx: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("reserve failed after lock release: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("reserve timed out after lock release")
	}
}
