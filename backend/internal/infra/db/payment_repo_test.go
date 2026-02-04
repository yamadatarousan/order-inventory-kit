package db

import "testing"

func TestPaymentRepository_確定と確認(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	_, err := db.Exec(`INSERT INTO orders (id, status) VALUES ($1, $2)`, "order-1", "accepted")
	if err != nil {
		t.Fatalf("failed to insert order: %v", err)
	}

	repo := NewPaymentRepository(db)
	if ok := repo.IsConfirmed("order-1", "k-1"); ok {
		t.Fatalf("expected false before confirm")
	}
	if err := repo.Confirm("order-1", "k-1"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ok := repo.IsConfirmed("order-1", "k-1"); !ok {
		t.Fatalf("expected true after confirm")
	}
}
