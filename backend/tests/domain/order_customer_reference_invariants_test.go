package domain_test

import (
	"errors"
	"testing"

	"order-inventory-kit/internal/domain"
	"order-inventory-kit/internal/usecase"
)

// このテストは 顧客参照整合の不変条件を固定する。
// 仕様対象: 未存在/無効 customerId の注文作成は失敗し、副作用を残さないこと。
// 根拠: 注文作成前提となる顧客参照が崩れた場合に、在庫/注文状態が汚染されないようにするため。
type 顧客参照不変条件用OrderRepo struct {
	createCalls int
}

func (r *顧客参照不変条件用OrderRepo) Create(_ domain.Order, _ map[string]int) error {
	r.createCalls++
	return nil
}

func (r *顧客参照不変条件用OrderRepo) Get(_ string) (domain.Order, bool) {
	return domain.Order{}, false
}

func (r *顧客参照不変条件用OrderRepo) GetTotalAmount(_ string) (int, bool) {
	return 0, false
}

func (r *顧客参照不変条件用OrderRepo) Update(_ domain.Order) error {
	return nil
}

type 顧客参照不変条件用PaymentRepo struct{}

func (r *顧客参照不変条件用PaymentRepo) IsConfirmed(_ string, _ string) bool {
	return false
}

func (r *顧客参照不変条件用PaymentRepo) ConfirmedAmount(_ string, _ string) (int, bool) {
	return 0, false
}

func (r *顧客参照不変条件用PaymentRepo) Confirm(_ string, _ string, _ int) error {
	return nil
}

type 顧客参照不変条件用InventoryRepo struct {
	inv          domain.Inventory
	reserveCalls int
	releaseCalls int
}

func new顧客参照不変条件用InventoryRepo(t *testing.T) *顧客参照不変条件用InventoryRepo {
	t.Helper()
	inv, err := domain.NewInventory("sku-1", 10, 0)
	if err != nil {
		t.Fatalf("failed to seed inventory: %v", err)
	}
	return &顧客参照不変条件用InventoryRepo{inv: inv}
}

func (r *顧客参照不変条件用InventoryRepo) Reserve(_ string, quantity int) (domain.Inventory, error) {
	r.reserveCalls++
	if err := r.inv.Reserve(quantity); err != nil {
		return domain.Inventory{}, err
	}
	return r.inv, nil
}

func (r *顧客参照不変条件用InventoryRepo) Release(_ string, quantity int) (domain.Inventory, error) {
	r.releaseCalls++
	if err := r.inv.Release(quantity); err != nil {
		return domain.Inventory{}, err
	}
	return r.inv, nil
}

type 顧客参照不変条件用CustomerRepo struct {
	states map[string]bool
}

func new顧客参照不変条件用CustomerRepo(states map[string]bool) *顧客参照不変条件用CustomerRepo {
	return &顧客参照不変条件用CustomerRepo{states: states}
}

func (r *顧客参照不変条件用CustomerRepo) IsActive(customerID string) (bool, error) {
	active, ok := r.states[customerID]
	if !ok {
		return false, nil
	}
	return active, nil
}

func TestCreateOrder_不変条件_未存在customerIdは失敗し副作用を残さない(t *testing.T) {
	orders := &顧客参照不変条件用OrderRepo{}
	payments := &顧客参照不変条件用PaymentRepo{}
	inventories := new顧客参照不変条件用InventoryRepo(t)
	customers := new顧客参照不変条件用CustomerRepo(map[string]bool{
		"c-1": true,
	})
	uc := usecase.NewOrderUsecase(orders, payments, inventories, customers, func() string { return "order-1" })

	_, err := uc.CreateOrder(usecase.CreateOrderInput{
		CustomerID:       "missing-customer",
		Items:            []domain.OrderItem{{SKU: "sku-1", Quantity: 1}},
		QuotedUnitPrices: map[string]int{"sku-1": 100},
	})
	if err == nil {
		t.Fatalf("missing customer must fail")
	}
	if !errors.Is(err, usecase.ErrCreateOrderInvalidCustomer) {
		t.Fatalf("expected ErrCreateOrderInvalidCustomer, got %v", err)
	}
	if orders.createCalls != 0 {
		t.Fatalf("order create side effect must not happen, got %d", orders.createCalls)
	}
	if inventories.reserveCalls != 0 {
		t.Fatalf("inventory reserve side effect must not happen, got %d", inventories.reserveCalls)
	}
	if inventories.releaseCalls != 0 {
		t.Fatalf("inventory release side effect must not happen, got %d", inventories.releaseCalls)
	}
	if inventories.inv.OnHand != 10 || inventories.inv.Reserved != 0 || inventories.inv.Available() != 10 {
		t.Fatalf("inventory must remain unchanged, got (%d,%d,%d)", inventories.inv.OnHand, inventories.inv.Reserved, inventories.inv.Available())
	}
}

func TestCreateOrder_不変条件_無効customerIdは失敗し副作用を残さない(t *testing.T) {
	orders := &顧客参照不変条件用OrderRepo{}
	payments := &顧客参照不変条件用PaymentRepo{}
	inventories := new顧客参照不変条件用InventoryRepo(t)
	customers := new顧客参照不変条件用CustomerRepo(map[string]bool{
		"c-1":               true,
		"inactive-customer": false,
	})
	uc := usecase.NewOrderUsecase(orders, payments, inventories, customers, func() string { return "order-1" })

	_, err := uc.CreateOrder(usecase.CreateOrderInput{
		CustomerID:       "inactive-customer",
		Items:            []domain.OrderItem{{SKU: "sku-1", Quantity: 1}},
		QuotedUnitPrices: map[string]int{"sku-1": 100},
	})
	if err == nil {
		t.Fatalf("inactive customer must fail")
	}
	if !errors.Is(err, usecase.ErrCreateOrderInvalidCustomer) {
		t.Fatalf("expected ErrCreateOrderInvalidCustomer, got %v", err)
	}
	if orders.createCalls != 0 {
		t.Fatalf("order create side effect must not happen, got %d", orders.createCalls)
	}
	if inventories.reserveCalls != 0 {
		t.Fatalf("inventory reserve side effect must not happen, got %d", inventories.reserveCalls)
	}
	if inventories.releaseCalls != 0 {
		t.Fatalf("inventory release side effect must not happen, got %d", inventories.releaseCalls)
	}
	if inventories.inv.OnHand != 10 || inventories.inv.Reserved != 0 || inventories.inv.Available() != 10 {
		t.Fatalf("inventory must remain unchanged, got (%d,%d,%d)", inventories.inv.OnHand, inventories.inv.Reserved, inventories.inv.Available())
	}
}
