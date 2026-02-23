package db

import "testing"

// このテストは CustomerRepository の顧客参照契約を固定する。
// 仕様対象: 存在かつ有効な顧客のみ IsActive=true を返すこと。
// 根拠: 注文作成前の顧客参照判定が実装差分で崩れないようにするため。
func TestCustomerRepository_IsActive_存在する有効顧客はtrue(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	repo := NewCustomerRepository(db)
	ok, err := repo.IsActive("c-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !ok {
		t.Fatalf("expected customer c-1 to be active")
	}
}

func TestCustomerRepository_IsActive_未存在顧客はfalse(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	repo := NewCustomerRepository(db)
	ok, err := repo.IsActive("missing-customer")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ok {
		t.Fatalf("expected missing customer to be inactive")
	}
}

func TestCustomerRepository_IsActive_無効顧客はfalse(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	if _, err := db.Exec(`UPDATE customers SET is_active = FALSE WHERE id = $1`, "c-1"); err != nil {
		t.Fatalf("failed to deactivate customer: %v", err)
	}

	repo := NewCustomerRepository(db)
	ok, err := repo.IsActive("c-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ok {
		t.Fatalf("expected inactive customer to be false")
	}
}
