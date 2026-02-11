package domain_test

import (
	"testing"

	"order-inventory-kit/internal/domain"
	"order-inventory-kit/internal/usecase"
)

// このテストは 注文確定の不変条件を固定する。
// 仕様対象: 再確定を禁止し、再確定要求時に状態と副作用が増えないこと。
// 根拠: 決済確定の重複実行で注文状態や課金記録が破壊されないようにするため。
type 確定不変条件用OrderRepo struct {
	order       domain.Order
	updateCalls int
}

func (r *確定不変条件用OrderRepo) Create(order domain.Order) error {
	r.order = order
	return nil
}

func (r *確定不変条件用OrderRepo) Get(id string) (domain.Order, bool) {
	if r.order.ID != id {
		return domain.Order{}, false
	}
	return r.order, true
}

func (r *確定不変条件用OrderRepo) Update(order domain.Order) error {
	r.updateCalls++
	r.order = order
	return nil
}

type 確定不変条件用PaymentRepo struct {
	keys         map[string]map[string]struct{}
	confirmCalls int
}

func new確定不変条件用PaymentRepo() *確定不変条件用PaymentRepo {
	return &確定不変条件用PaymentRepo{
		keys: make(map[string]map[string]struct{}),
	}
}

func (r *確定不変条件用PaymentRepo) IsConfirmed(orderID, idempotencyKey string) bool {
	perOrder, ok := r.keys[orderID]
	if !ok {
		return false
	}
	_, exists := perOrder[idempotencyKey]
	return exists
}

func (r *確定不変条件用PaymentRepo) Confirm(orderID, idempotencyKey string) error {
	perOrder, ok := r.keys[orderID]
	if !ok {
		perOrder = make(map[string]struct{})
		r.keys[orderID] = perOrder
	}
	perOrder[idempotencyKey] = struct{}{}
	r.confirmCalls++
	return nil
}

func (r *確定不変条件用PaymentRepo) confirmedCount(orderID string) int {
	return len(r.keys[orderID])
}

func TestConfirmPayment_不変条件_再確定は失敗し状態と副作用が増えない(t *testing.T) {
	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 1}})
	orders := &確定不変条件用OrderRepo{order: order}
	payments := new確定不変条件用PaymentRepo()
	uc := usecase.NewOrderUsecase(orders, payments, func() string { return "unused" })

	_, err := uc.ConfirmPayment(usecase.ConfirmPaymentInput{
		OrderID:        "order-1",
		Amount:         100,
		IdempotencyKey: "k-1",
	})
	if err != nil {
		t.Fatalf("first confirm must succeed: %v", err)
	}

	if orders.order.Status != domain.OrderStatusConfirmed {
		t.Fatalf("expected confirmed, got %s", orders.order.Status)
	}
	if orders.updateCalls != 1 {
		t.Fatalf("expected one order update, got %d", orders.updateCalls)
	}
	if payments.confirmCalls != 1 {
		t.Fatalf("expected one payment confirm, got %d", payments.confirmCalls)
	}
	if payments.confirmedCount("order-1") != 1 {
		t.Fatalf("expected one payment key, got %d", payments.confirmedCount("order-1"))
	}

	_, err = uc.ConfirmPayment(usecase.ConfirmPaymentInput{
		OrderID:        "order-1",
		Amount:         100,
		IdempotencyKey: "k-2",
	})
	if err == nil {
		t.Fatalf("second confirm must fail")
	}

	if orders.order.Status != domain.OrderStatusConfirmed {
		t.Fatalf("status must remain confirmed, got %s", orders.order.Status)
	}
	if orders.updateCalls != 1 {
		t.Fatalf("order update side effect must not increase, got %d", orders.updateCalls)
	}
	if payments.confirmCalls != 1 {
		t.Fatalf("payment confirm side effect must not increase, got %d", payments.confirmCalls)
	}
	if payments.confirmedCount("order-1") != 1 {
		t.Fatalf("payment keys must not increase, got %d", payments.confirmedCount("order-1"))
	}
}

func TestConfirmPayment_不変条件_同一キー再送で支払いが二重計上されない(t *testing.T) {
	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 1}})
	orders := &確定不変条件用OrderRepo{order: order}
	payments := new確定不変条件用PaymentRepo()
	uc := usecase.NewOrderUsecase(orders, payments, func() string { return "unused" })

	_, err := uc.ConfirmPayment(usecase.ConfirmPaymentInput{
		OrderID:        "order-1",
		Amount:         100,
		IdempotencyKey: "k-1",
	})
	if err != nil {
		t.Fatalf("first confirm must succeed: %v", err)
	}

	out, err := uc.ConfirmPayment(usecase.ConfirmPaymentInput{
		OrderID:        "order-1",
		Amount:         100,
		IdempotencyKey: "k-1",
	})
	if err != nil {
		t.Fatalf("same-key replay must be idempotent: %v", err)
	}
	if out.PaymentStatus != "confirmed" {
		t.Fatalf("expected confirmed, got %s", out.PaymentStatus)
	}
	if orders.order.Status != domain.OrderStatusConfirmed {
		t.Fatalf("status must remain confirmed, got %s", orders.order.Status)
	}
	if orders.updateCalls != 1 {
		t.Fatalf("order update side effect must not increase, got %d", orders.updateCalls)
	}
	if payments.confirmCalls != 1 {
		t.Fatalf("payment confirm side effect must not increase, got %d", payments.confirmCalls)
	}
	if payments.confirmedCount("order-1") != 1 {
		t.Fatalf("payment keys must not increase, got %d", payments.confirmedCount("order-1"))
	}
}
