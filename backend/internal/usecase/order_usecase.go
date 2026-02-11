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
	Update(order domain.Order) error
}

// PaymentRepository は決済の冪等性記録を抽象化する。
type PaymentRepository interface {
	IsConfirmed(orderID, idempotencyKey string) bool
	Confirm(orderID, idempotencyKey string) error
}

// OrderUsecase は注文まわりのユースケースを提供する。
type OrderUsecase struct {
	orders   OrderRepository
	payments PaymentRepository
	idGen    func() string
}

// NewOrderUsecase は依存を受け取ってユースケースを生成する。
func NewOrderUsecase(orders OrderRepository, payments PaymentRepository, idGen func() string) *OrderUsecase {
	return &OrderUsecase{orders: orders, payments: payments, idGen: idGen}
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
	order, err := domain.NewOrder(u.idGen(), input.Items)
	if err != nil {
		return CreateOrderOutput{}, err
	}
	if err := u.orders.Create(order); err != nil {
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
	order, ok := u.orders.Get(id)
	if !ok {
		return domain.Order{}, errors.New("not found")
	}
	order.Status = domain.OrderStatusCanceled
	if err := u.orders.Update(order); err != nil {
		return domain.Order{}, err
	}
	return order, nil
}

// ConfirmPaymentInput は決済確定の入力。
type ConfirmPaymentInput struct {
	OrderID        string
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
		return ConfirmPaymentOutput{OrderID: input.OrderID, PaymentStatus: "confirmed"}, nil
	}
	if order.Status == domain.OrderStatusConfirmed {
		return ConfirmPaymentOutput{}, errors.New("already confirmed")
	}
	order.Status = domain.OrderStatusConfirmed
	if err := u.orders.Update(order); err != nil {
		return ConfirmPaymentOutput{}, err
	}
	if err := u.payments.Confirm(input.OrderID, input.IdempotencyKey); err != nil {
		return ConfirmPaymentOutput{}, err
	}
	return ConfirmPaymentOutput{OrderID: input.OrderID, PaymentStatus: "confirmed"}, nil
}
