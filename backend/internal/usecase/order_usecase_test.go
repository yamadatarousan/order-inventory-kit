package usecase

import (
	"testing"

	"order-inventory-kit/internal/domain"
)

// このテストは OrderUsecase の主要ユースケース仕様を固定する。
// 仕様対象: 注文作成/参照/キャンセル/決済確定の状態遷移と冪等性。
// 根拠: 下位実装変更時にも業務フローの期待挙動を維持するため。
type memoryOrderRepo struct {
	items map[string]domain.Order
}

func newMemoryOrderRepo() *memoryOrderRepo {
	return &memoryOrderRepo{items: make(map[string]domain.Order)}
}

func (r *memoryOrderRepo) Create(order domain.Order) error {
	r.items[order.ID] = order
	return nil
}

func (r *memoryOrderRepo) Get(id string) (domain.Order, bool) {
	order, ok := r.items[id]
	return order, ok
}

func (r *memoryOrderRepo) Update(order domain.Order) error {
	r.items[order.ID] = order
	return nil
}

type memoryPaymentRepo struct {
	keys map[string]map[string]struct{}
}

func newMemoryPaymentRepo() *memoryPaymentRepo {
	return &memoryPaymentRepo{keys: make(map[string]map[string]struct{})}
}

func (r *memoryPaymentRepo) IsConfirmed(orderID, idempotencyKey string) bool {
	keys, ok := r.keys[orderID]
	if !ok {
		return false
	}
	_, exists := keys[idempotencyKey]
	return exists
}

func (r *memoryPaymentRepo) Confirm(orderID, idempotencyKey string) error {
	keys, ok := r.keys[orderID]
	if !ok {
		keys = make(map[string]struct{})
		r.keys[orderID] = keys
	}
	keys[idempotencyKey] = struct{}{}
	return nil
}

func TestCreateOrder_正常系(t *testing.T) {
	orders := newMemoryOrderRepo()
	payments := newMemoryPaymentRepo()
	uc := NewOrderUsecase(orders, payments, func() string { return "order-1" })

	out, err := uc.CreateOrder(CreateOrderInput{CustomerID: "c-1", Items: []domain.OrderItem{{SKU: "sku-1", Quantity: 1}}})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.OrderID != "order-1" {
		t.Fatalf("expected order-1, got %s", out.OrderID)
	}
	if out.Status != domain.OrderStatusAccepted {
		t.Fatalf("expected accepted, got %s", out.Status)
	}
	if _, ok := orders.Get("order-1"); !ok {
		t.Fatalf("expected order to be saved")
	}
}

func TestCreateOrder_異常系_不正な入力(t *testing.T) {
	orders := newMemoryOrderRepo()
	payments := newMemoryPaymentRepo()
	uc := NewOrderUsecase(orders, payments, func() string { return "order-1" })

	_, err := uc.CreateOrder(CreateOrderInput{CustomerID: "c-1", Items: []domain.OrderItem{}})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestGetOrder_存在する場合(t *testing.T) {
	orders := newMemoryOrderRepo()
	payments := newMemoryPaymentRepo()
	uc := NewOrderUsecase(orders, payments, func() string { return "order-1" })

	order, _ := domain.NewOrder("order-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 1}})
	_ = orders.Create(order)

	got, ok := uc.GetOrder("order-1")
	if !ok {
		t.Fatalf("expected order to exist")
	}
	if got.ID != "order-1" {
		t.Fatalf("expected order-1, got %s", got.ID)
	}
}

func TestGetOrder_存在しない場合(t *testing.T) {
	orders := newMemoryOrderRepo()
	payments := newMemoryPaymentRepo()
	uc := NewOrderUsecase(orders, payments, func() string { return "order-1" })

	_, ok := uc.GetOrder("missing")
	if ok {
		t.Fatalf("expected not found")
	}
}

func TestCancelOrder_正常系(t *testing.T) {
	orders := newMemoryOrderRepo()
	payments := newMemoryPaymentRepo()
	uc := NewOrderUsecase(orders, payments, func() string { return "order-1" })

	order, _ := domain.NewOrder("order-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 1}})
	_ = orders.Create(order)

	canceled, err := uc.CancelOrder("order-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if canceled.Status != domain.OrderStatusCanceled {
		t.Fatalf("expected canceled, got %s", canceled.Status)
	}
}

func TestCancelOrder_存在しない場合(t *testing.T) {
	orders := newMemoryOrderRepo()
	payments := newMemoryPaymentRepo()
	uc := NewOrderUsecase(orders, payments, func() string { return "order-1" })

	_, err := uc.CancelOrder("missing")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestConfirmPayment_正常系(t *testing.T) {
	orders := newMemoryOrderRepo()
	payments := newMemoryPaymentRepo()
	uc := NewOrderUsecase(orders, payments, func() string { return "order-1" })

	order, _ := domain.NewOrder("order-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 1}})
	_ = orders.Create(order)

	out, err := uc.ConfirmPayment(ConfirmPaymentInput{OrderID: "order-1", Amount: 100, IdempotencyKey: "k-1"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.PaymentStatus != "confirmed" {
		t.Fatalf("expected confirmed, got %s", out.PaymentStatus)
	}
	saved, _ := orders.Get("order-1")
	if saved.Status != domain.OrderStatusConfirmed {
		t.Fatalf("expected confirmed status, got %s", saved.Status)
	}
}

func TestConfirmPayment_異常系_不正な入力(t *testing.T) {
	orders := newMemoryOrderRepo()
	payments := newMemoryPaymentRepo()
	uc := NewOrderUsecase(orders, payments, func() string { return "order-1" })

	_, err := uc.ConfirmPayment(ConfirmPaymentInput{OrderID: "", Amount: 0, IdempotencyKey: ""})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestConfirmPayment_異常系_注文が存在しない(t *testing.T) {
	orders := newMemoryOrderRepo()
	payments := newMemoryPaymentRepo()
	uc := NewOrderUsecase(orders, payments, func() string { return "order-1" })

	_, err := uc.ConfirmPayment(ConfirmPaymentInput{OrderID: "missing", Amount: 100, IdempotencyKey: "k-1"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestConfirmPayment_冪等(t *testing.T) {
	orders := newMemoryOrderRepo()
	payments := newMemoryPaymentRepo()
	uc := NewOrderUsecase(orders, payments, func() string { return "order-1" })

	order, _ := domain.NewOrder("order-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 1}})
	_ = orders.Create(order)

	_, _ = uc.ConfirmPayment(ConfirmPaymentInput{OrderID: "order-1", Amount: 100, IdempotencyKey: "k-1"})
	out, err := uc.ConfirmPayment(ConfirmPaymentInput{OrderID: "order-1", Amount: 100, IdempotencyKey: "k-1"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.PaymentStatus != "confirmed" {
		t.Fatalf("expected confirmed, got %s", out.PaymentStatus)
	}
}
