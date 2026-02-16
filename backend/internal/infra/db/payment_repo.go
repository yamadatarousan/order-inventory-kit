// Package db はDB実装を提供する。
package db

import "database/sql"

// PaymentRepository はDBの決済リポジトリ。
type PaymentRepository struct {
	db *sql.DB
}

// NewPaymentRepository はDBの決済リポジトリを作成する。
func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// IsConfirmed は確定済みかを判定する。
func (r *PaymentRepository) IsConfirmed(orderID, idempotencyKey string) bool {
	row := r.db.QueryRow(`
		SELECT 1 FROM payments WHERE order_id = $1 AND idempotency_key = $2
	`, orderID, idempotencyKey)
	var v int
	if err := row.Scan(&v); err != nil {
		return false
	}
	return true
}

// Confirm は決済を確定する。
func (r *PaymentRepository) Confirm(orderID, idempotencyKey string, amount int) error {
	_, err := r.db.Exec(`
		INSERT INTO payments (order_id, idempotency_key, status, amount) VALUES ($1, $2, 'confirmed', $3)
	`, orderID, idempotencyKey, amount)
	return err
}
