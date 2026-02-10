package domain

import "errors"

// Inventory はSKU単位の在庫を表す。
type Inventory struct {
	SKU      string
	Quantity int
}

// NewInventory は入力を検証して Inventory を作成する。
func NewInventory(sku string, quantity int) (Inventory, error) {
	if sku == "" {
		return Inventory{}, errors.New("sku is required")
	}
	if quantity < 0 {
		return Inventory{}, errors.New("quantity must be >= 0")
	}
	return Inventory{SKU: sku, Quantity: quantity}, nil
}

// Reserve は在庫を確保する。
func (i *Inventory) Reserve(quantity int) error {
	if quantity < 1 {
		return errors.New("reserve quantity must be >= 1")
	}
	if i.Quantity < quantity {
		return errors.New("insufficient inventory")
	}
	i.Quantity -= quantity
	return nil
}

// Release は在庫を戻す。
func (i *Inventory) Release(quantity int) error {
	if quantity < 1 {
		return errors.New("release quantity must be >= 1")
	}
	i.Quantity += quantity
	return nil
}
