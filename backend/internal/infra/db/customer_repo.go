// Package db はDB実装を提供する。
package db

import "database/sql"

// CustomerRepository は顧客マスタ参照のDB実装。
type CustomerRepository struct {
	db *sql.DB
}

// NewCustomerRepository は顧客マスタ参照リポジトリを作成する。
func NewCustomerRepository(db *sql.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

// IsActive は顧客IDが存在し、かつ有効かを返す。
func (r *CustomerRepository) IsActive(customerID string) (bool, error) {
	var exists bool
	if err := r.db.QueryRow(`
		SELECT EXISTS(
		  SELECT 1
		  FROM customers
		  WHERE id = $1 AND is_active = TRUE
		)
	`, customerID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}
