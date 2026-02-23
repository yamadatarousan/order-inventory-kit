// Package usecase はユースケース層を定義する。
package usecase

import (
	"errors"

	"order-inventory-kit/internal/domain"
)

// OrderRepository は注文の永続化を抽象化する。
type OrderRepository interface {
	Create(order domain.Order) error
	Get(id string) (domain.Order, bool)
	GetTotalAmount(id string) (int, bool)
	Update(order domain.Order) error
}

// PaymentRepository は決済の冪等性記録を抽象化する。
type PaymentRepository interface {
	IsConfirmed(orderID, idempotencyKey string) bool
	ConfirmedAmount(orderID, idempotencyKey string) (int, bool)
	Confirm(orderID, idempotencyKey string, amount int) error
}

// OrderInventoryRepository は注文操作で使う在庫更新を抽象化する。
type OrderInventoryRepository interface {
	Reserve(sku string, quantity int) (domain.Inventory, error)
	Release(sku string, quantity int) (domain.Inventory, error)
}

// OrderUsecase は注文まわりのユースケースを提供する。
type OrderUsecase struct {
	orders      OrderRepository
	payments    PaymentRepository
	inventories OrderInventoryRepository
	idGen       func() string
}

// NewOrderUsecase は依存を受け取ってユースケースを生成する。
func NewOrderUsecase(
	orders OrderRepository,
	payments PaymentRepository,
	inventories OrderInventoryRepository,
	idGen func() string,
) *OrderUsecase {
	return &OrderUsecase{orders: orders, payments: payments, inventories: inventories, idGen: idGen}
}

// CreateOrderInput は注文作成の入力。
type CreateOrderInput struct {
	CustomerID string
	Items      []domain.OrderItem
}

// CreateOrderOutput は注文作成の出力。
type CreateOrderOutput struct {
	OrderID string
	Status  domain.OrderStatus
}

// CreateOrder は注文を作成する。
func (u *OrderUsecase) CreateOrder(input CreateOrderInput) (CreateOrderOutput, error) {
	if u.inventories == nil {
		return CreateOrderOutput{}, errors.New("inventory repository is required")
	}

	order, err := domain.NewOrder(u.idGen(), input.CustomerID, input.Items)
	if err != nil {
		return CreateOrderOutput{}, err
	}

	reservedItems := make([]domain.OrderItem, 0, len(order.Items))
	for _, item := range order.Items {
		if _, err := u.inventories.Reserve(item.SKU, item.Quantity); err != nil {
			if compensationErr := u.compensateRelease(reservedItems); compensationErr != nil {
				return CreateOrderOutput{}, errors.Join(errors.New("compensation failed"), err, compensationErr)
			}
			return CreateOrderOutput{}, err
		}
		reservedItems = append(reservedItems, item)
	}

	if err := u.orders.Create(order); err != nil {
		if compensationErr := u.compensateRelease(reservedItems); compensationErr != nil {
			return CreateOrderOutput{}, errors.Join(errors.New("compensation failed"), err, compensationErr)
		}
		return CreateOrderOutput{}, err
	}
	return CreateOrderOutput{OrderID: order.ID, Status: order.Status}, nil
}

// GetOrder は注文を取得する。
func (u *OrderUsecase) GetOrder(id string) (domain.Order, bool) {
	return u.orders.Get(id)
}

// CancelOrder は注文をキャンセルする。
func (u *OrderUsecase) CancelOrder(id string) (domain.Order, error) {
	if u.inventories == nil {
		return domain.Order{}, errors.New("inventory repository is required")
	}

	order, ok := u.orders.Get(id)
	if !ok {
		return domain.Order{}, errors.New("not found")
	}
	if order.Status == domain.OrderStatusCanceled {
		return domain.Order{}, errors.New("already canceled")
	}

	releasedItems := make([]domain.OrderItem, 0, len(order.Items))
	for _, item := range order.Items {
		if _, err := u.inventories.Release(item.SKU, item.Quantity); err != nil {
			if compensationErr := u.compensateReserve(releasedItems); compensationErr != nil {
				return domain.Order{}, errors.Join(errors.New("compensation failed"), err, compensationErr)
			}
			return domain.Order{}, err
		}
		releasedItems = append(releasedItems, item)
	}

	order.Status = domain.OrderStatusCanceled
	if err := u.orders.Update(order); err != nil {
		if compensationErr := u.compensateReserve(releasedItems); compensationErr != nil {
			return domain.Order{}, errors.Join(errors.New("compensation failed"), err, compensationErr)
		}
		return domain.Order{}, err
	}
	return order, nil
}

// ConfirmPaymentInput は決済確定の入力。
type ConfirmPaymentInput struct {
	OrderID string
	// Amount は決済要求額（最小通貨単位）。
	Amount         int
	IdempotencyKey string
}

// ConfirmPaymentOutput は決済確定の出力。
type ConfirmPaymentOutput struct {
	OrderID       string
	PaymentStatus string
}

// ConfirmPayment は決済を確定する。
func (u *OrderUsecase) ConfirmPayment(input ConfirmPaymentInput) (ConfirmPaymentOutput, error) {
	if input.OrderID == "" || input.IdempotencyKey == "" || input.Amount < 1 {
		return ConfirmPaymentOutput{}, errors.New("invalid request")
	}
	order, ok := u.orders.Get(input.OrderID)
	if !ok {
		return ConfirmPaymentOutput{}, errors.New("not found")
	}
	if u.payments.IsConfirmed(input.OrderID, input.IdempotencyKey) {
		confirmedAmount, found := u.payments.ConfirmedAmount(input.OrderID, input.IdempotencyKey)
		if found && confirmedAmount != input.Amount {
			return ConfirmPaymentOutput{}, errors.New("amount conflict")
		}
		return ConfirmPaymentOutput{OrderID: input.OrderID, PaymentStatus: "confirmed"}, nil
	}
	totalAmount, ok := u.orders.GetTotalAmount(input.OrderID)
	if !ok {
		return ConfirmPaymentOutput{}, errors.New("not found")
	}
	if input.Amount != totalAmount {
		return ConfirmPaymentOutput{}, errors.New("amount conflict")
	}
	if order.Status == domain.OrderStatusConfirmed {
		return ConfirmPaymentOutput{}, errors.New("already confirmed")
	}
	order.Status = domain.OrderStatusConfirmed
	if err := u.orders.Update(order); err != nil {
		return ConfirmPaymentOutput{}, err
	}
	if err := u.payments.Confirm(input.OrderID, input.IdempotencyKey, input.Amount); err != nil {
		return ConfirmPaymentOutput{}, err
	}
	return ConfirmPaymentOutput{OrderID: input.OrderID, PaymentStatus: "confirmed"}, nil
}

func (u *OrderUsecase) compensateRelease(items []domain.OrderItem) error {
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		if _, err := u.inventories.Release(item.SKU, item.Quantity); err != nil {
			return err
		}
	}
	return nil
}

func (u *OrderUsecase) compensateReserve(items []domain.OrderItem) error {
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		if _, err := u.inventories.Reserve(item.SKU, item.Quantity); err != nil {
			return err
		}
	}
	return nil
}
