package usecase

import (
	"errors"

	"order-inventory-kit/internal/domain"
)

// InventoryRepository は在庫の永続化を抽象化する。
type InventoryRepository interface {
	GetBySKU(sku string) (domain.Inventory, bool)
	Update(inventory domain.Inventory) error
}

// InventoryUsecase は在庫ユースケースを提供する。
type InventoryUsecase struct {
	inventories InventoryRepository
}

// NewInventoryUsecase は在庫ユースケースを作成する。
func NewInventoryUsecase(inventories InventoryRepository) *InventoryUsecase {
	return &InventoryUsecase{inventories: inventories}
}

// ReserveInventoryInput は在庫確保の入力。
type ReserveInventoryInput struct {
	SKU      string
	Quantity int
}

// ReserveInventoryOutput は在庫確保の出力。
type ReserveInventoryOutput struct {
	SKU               string
	RemainingQuantity int
}

// ReserveInventory は在庫を確保する。
func (u *InventoryUsecase) ReserveInventory(input ReserveInventoryInput) (ReserveInventoryOutput, error) {
	if input.SKU == "" || input.Quantity < 1 {
		return ReserveInventoryOutput{}, errors.New("invalid request")
	}

	inventory, ok := u.inventories.GetBySKU(input.SKU)
	if !ok {
		return ReserveInventoryOutput{}, errors.New("not found")
	}

	if err := inventory.Reserve(input.Quantity); err != nil {
		return ReserveInventoryOutput{}, err
	}

	if err := u.inventories.Update(inventory); err != nil {
		return ReserveInventoryOutput{}, err
	}

	return ReserveInventoryOutput{
		SKU:               inventory.SKU,
		RemainingQuantity: inventory.Quantity,
	}, nil
}

// ReleaseInventoryInput は在庫戻しの入力。
type ReleaseInventoryInput struct {
	SKU      string
	Quantity int
}

// ReleaseInventoryOutput は在庫戻しの出力。
type ReleaseInventoryOutput struct {
	SKU      string
	Quantity int
}

// ReleaseInventory は在庫を戻す。
func (u *InventoryUsecase) ReleaseInventory(input ReleaseInventoryInput) (ReleaseInventoryOutput, error) {
	if input.SKU == "" || input.Quantity < 1 {
		return ReleaseInventoryOutput{}, errors.New("invalid request")
	}

	inventory, ok := u.inventories.GetBySKU(input.SKU)
	if !ok {
		return ReleaseInventoryOutput{}, errors.New("not found")
	}

	if err := inventory.Release(input.Quantity); err != nil {
		return ReleaseInventoryOutput{}, err
	}

	if err := u.inventories.Update(inventory); err != nil {
		return ReleaseInventoryOutput{}, err
	}

	return ReleaseInventoryOutput{
		SKU:      inventory.SKU,
		Quantity: inventory.Quantity,
	}, nil
}
