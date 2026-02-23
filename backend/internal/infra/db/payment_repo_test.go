package db

import (
	"database/sql"
	"testing"
)

// このテストは PaymentRepository の確定判定仕様を固定する。
// 仕様対象: Confirm 前後で IsConfirmed の結果が期待どおりに変化すること。
// 根拠: 決済の冪等制御に必要な判定結果が実装変更で崩れないようにするため。
func TestPaymentRepository_確定と確認(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	_, err := db.Exec(`INSERT INTO orders (id, customer_id, status) VALUES ($1, $2, $3)`, "order-1", "c-1", "accepted")
	if err != nil {
		t.Fatalf("failed to insert order: %v", err)
	}

	repo := NewPaymentRepository(db)
	if ok := repo.IsConfirmed("order-1", "k-1"); ok {
		t.Fatalf("expected false before confirm")
	}
	if err := repo.Confirm("order-1", "k-1", 100); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ok := repo.IsConfirmed("order-1", "k-1"); !ok {
		t.Fatalf("expected true after confirm")
	}
	confirmedAmount, ok := repo.ConfirmedAmount("order-1", "k-1")
	if !ok {
		t.Fatalf("expected confirmed amount to exist")
	}
	if confirmedAmount != 100 {
		t.Fatalf("expected confirmed amount=100, got %d", confirmedAmount)
	}

	var amount sql.NullInt64
	if err := db.QueryRow(`
		SELECT amount FROM payments WHERE order_id = $1 AND idempotency_key = $2
	`, "order-1", "k-1").Scan(&amount); err != nil {
		t.Fatalf("failed to load payment amount: %v", err)
	}
	if !amount.Valid {
		t.Fatalf("expected payment amount to be set")
	}
	if amount.Int64 != 100 {
		t.Fatalf("expected payment amount=100, got %d", amount.Int64)
	}
}
