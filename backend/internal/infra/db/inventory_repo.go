package db

import (
	"context"
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
		SELECT sku, on_hand, reserved FROM inventory WHERE sku = $1
	`, sku)

	var (
		invSKU   string
		onHand   int
		reserved int
	)
	if err := row.Scan(&invSKU, &onHand, &reserved); err != nil {
		return domain.Inventory{}, false
	}
	inv, err := domain.NewInventory(invSKU, onHand, reserved)
	if err != nil {
		return domain.Inventory{}, false
	}
	return inv, true
}

// Reserve は在庫を確保する。
func (r *InventoryRepository) Reserve(sku string, quantity int) (domain.Inventory, error) {
	if quantity < 1 {
		return domain.Inventory{}, errors.New("invalid quantity")
	}

	tx, err := r.db.BeginTx(context.Background(), nil)
	if err != nil {
		return domain.Inventory{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var (
		invSKU   string
		onHand   int
		reserved int
	)
	row := tx.QueryRow(`
		SELECT sku, on_hand, reserved FROM inventory WHERE sku = $1 FOR UPDATE
	`, sku)
	if err := row.Scan(&invSKU, &onHand, &reserved); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Inventory{}, errors.New("not found")
		}
		return domain.Inventory{}, err
	}
	inv, err := domain.NewInventory(invSKU, onHand, reserved)
	if err != nil {
		return domain.Inventory{}, err
	}

	if err := inv.Reserve(quantity); err != nil {
		return domain.Inventory{}, err
	}

	if _, err := tx.Exec(`
		UPDATE inventory
		SET on_hand = $2, reserved = $3, quantity = $4
		WHERE sku = $1
	`, inv.SKU, inv.OnHand, inv.Reserved, inv.Available()); err != nil {
		return domain.Inventory{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Inventory{}, err
	}
	committed = true
	return inv, nil
}

// Release は在庫を戻す。
func (r *InventoryRepository) Release(sku string, quantity int) (domain.Inventory, error) {
	if quantity < 1 {
		return domain.Inventory{}, errors.New("invalid quantity")
	}

	tx, err := r.db.BeginTx(context.Background(), nil)
	if err != nil {
		return domain.Inventory{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var (
		invSKU   string
		onHand   int
		reserved int
	)
	row := tx.QueryRow(`
		SELECT sku, on_hand, reserved FROM inventory WHERE sku = $1 FOR UPDATE
	`, sku)
	if err := row.Scan(&invSKU, &onHand, &reserved); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Inventory{}, errors.New("not found")
		}
		return domain.Inventory{}, err
	}
	inv, err := domain.NewInventory(invSKU, onHand, reserved)
	if err != nil {
		return domain.Inventory{}, err
	}

	if err := inv.Release(quantity); err != nil {
		return domain.Inventory{}, err
	}

	if _, err := tx.Exec(`
		UPDATE inventory
		SET on_hand = $2, reserved = $3, quantity = $4
		WHERE sku = $1
	`, inv.SKU, inv.OnHand, inv.Reserved, inv.Available()); err != nil {
		return domain.Inventory{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Inventory{}, err
	}
	committed = true
	return inv, nil
}
