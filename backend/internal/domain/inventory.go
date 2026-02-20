package domain

import "errors"

// Inventory はSKU単位の在庫を表す。
type Inventory struct {
	SKU      string
	OnHand   int
	Reserved int
	// Quantity は旧モデル互換のための値。Available と同値を維持する。
	Quantity int
}

// NewInventory は入力を検証して Inventory を作成する。
func NewInventory(sku string, values ...int) (Inventory, error) {
	var (
		onHand   int
		reserved int
	)
	switch len(values) {
	case 1:
		onHand = values[0]
	case 2:
		onHand = values[0]
		reserved = values[1]
	default:
		return Inventory{}, errors.New("inventory requires (sku, quantity) or (sku, on_hand, reserved)")
	}

	if sku == "" {
		return Inventory{}, errors.New("sku is required")
	}
	if onHand < 0 {
		return Inventory{}, errors.New("on hand must be >= 0")
	}
	if reserved < 0 {
		return Inventory{}, errors.New("reserved must be >= 0")
	}
	if onHand < reserved {
		return Inventory{}, errors.New("on hand must be >= reserved")
	}

	inv := Inventory{
		SKU:      sku,
		OnHand:   onHand,
		Reserved: reserved,
	}
	inv.syncQuantity()
	return inv, nil
}

// Available は販売可能在庫（OnHand - Reserved）を返す。
func (i Inventory) Available() int {
	return i.OnHand - i.Reserved
}

// Reserve は在庫を確保する。
func (i *Inventory) Reserve(quantity int) error {
	if quantity < 1 {
		return errors.New("reserve quantity must be >= 1")
	}

	if i.Available() < quantity {
		return errors.New("insufficient inventory")
	}
	i.Reserved += quantity
	i.syncQuantity()
	return nil
}

// Release は在庫を戻す。
func (i *Inventory) Release(quantity int) error {
	if quantity < 1 {
		return errors.New("release quantity must be >= 1")
	}
	if i.Reserved < quantity {
		return errors.New("insufficient reserved inventory")
	}
	i.Reserved -= quantity
	i.syncQuantity()
	return nil
}

func (i *Inventory) syncQuantity() {
	i.Quantity = i.Available()
}
