package db

import (
	"database/sql"
	"errors"

	"order-inventory-kit/internal/domain"
)

// InventoryRepository はDBの在庫リポジトリ。
type InventoryRepository struct {
	db *sql.DB
}

// NewInventoryRepository はDBの在庫リポジトリを作成する。
func NewInventoryRepository(db *sql.DB) *InventoryRepository {
	return &InventoryRepository{db: db}
}

// GetBySKU はSKUで在庫を取得する。
func (r *InventoryRepository) GetBySKU(sku string) (domain.Inventory, bool) {
	row := r.db.QueryRow(`
		SELECT sku, quantity FROM inventory WHERE sku = $1
	`, sku)

	var inv domain.Inventory
	if err := row.Scan(&inv.SKU, &inv.Quantity); err != nil {
		return domain.Inventory{}, false
	}
	return inv, true
}

// Reserve は在庫を確保する。
func (r *InventoryRepository) Reserve(sku string, quantity int) (domain.Inventory, error) {
	if quantity < 1 {
		return domain.Inventory{}, errors.New("invalid quantity")
	}

	row := r.db.QueryRow(`
		UPDATE inventory
		SET quantity = quantity - $2
		WHERE sku = $1 AND quantity >= $2
		RETURNING sku, quantity
	`, sku, quantity)

	var inv domain.Inventory
	if err := row.Scan(&inv.SKU, &inv.Quantity); err == nil {
		return inv, nil
	}

	if _, ok := r.GetBySKU(sku); !ok {
		return domain.Inventory{}, errors.New("not found")
	}
	return domain.Inventory{}, errors.New("insufficient inventory")
}

// Release は在庫を戻す。
func (r *InventoryRepository) Release(sku string, quantity int) (domain.Inventory, error) {
	if quantity < 1 {
		return domain.Inventory{}, errors.New("invalid quantity")
	}

	row := r.db.QueryRow(`
		UPDATE inventory
		SET quantity = quantity + $2
		WHERE sku = $1
		RETURNING sku, quantity
	`, sku, quantity)

	var inv domain.Inventory
	if err := row.Scan(&inv.SKU, &inv.Quantity); err != nil {
		return domain.Inventory{}, errors.New("not found")
	}
	return inv, nil
}
