package usecase

import (
	"errors"
	"strings"
	"testing"

	"order-inventory-kit/internal/domain"
)

// このテストは OrderUsecase の主要ユースケース仕様を固定する。
// 仕様対象: 注文作成/参照/キャンセル/決済確定の状態遷移と在庫接続時の補償挙動。
// 根拠: 下位実装変更時にも業務フローの期待挙動と副作用制御を維持するため。
type memoryOrderRepo struct {
	items       map[string]domain.Order
	createCalls int
	updateCalls int
	failCreate  bool
	failUpdate  bool
}

func newMemoryOrderRepo() *memoryOrderRepo {
	return &memoryOrderRepo{items: make(map[string]domain.Order)}
}

func (r *memoryOrderRepo) Create(order domain.Order) error {
	r.createCalls++
	if r.failCreate {
		return errors.New("create failed")
	}
	r.items[order.ID] = order
	return nil
}

func (r *memoryOrderRepo) Get(id string) (domain.Order, bool) {
	order, ok := r.items[id]
	return order, ok
}

func (r *memoryOrderRepo) GetTotalAmount(id string) (int, bool) {
	order, ok := r.items[id]
	if !ok {
		return 0, false
	}
	total := 0
	for _, item := range order.Items {
		// usecase単体テストでは価格源泉を固定していないため、1個あたり100円で合計を計算する。
		total += item.Quantity * 100
	}
	return total, true
}

func (r *memoryOrderRepo) Update(order domain.Order) error {
	r.updateCalls++
	if r.failUpdate {
		return errors.New("update failed")
	}
	r.items[order.ID] = order
	return nil
}

type memoryPaymentRepo struct {
	keys map[string]map[string]int
}

func newMemoryPaymentRepo() *memoryPaymentRepo {
	return &memoryPaymentRepo{keys: make(map[string]map[string]int)}
}

func (r *memoryPaymentRepo) IsConfirmed(orderID, idempotencyKey string) bool {
	keys, ok := r.keys[orderID]
	if !ok {
		return false
	}
	_, exists := keys[idempotencyKey]
	return exists
}

func (r *memoryPaymentRepo) ConfirmedAmount(orderID, idempotencyKey string) (int, bool) {
	keys, ok := r.keys[orderID]
	if !ok {
		return 0, false
	}
	amount, exists := keys[idempotencyKey]
	return amount, exists
}

func (r *memoryPaymentRepo) Confirm(orderID, idempotencyKey string, amount int) error {
	keys, ok := r.keys[orderID]
	if !ok {
		keys = make(map[string]int)
		r.keys[orderID] = keys
	}
	keys[idempotencyKey] = amount
	return nil
}

type memoryOrderInventoryRepo struct {
	items        map[string]domain.Inventory
	reserveCalls int
	releaseCalls int
	failReserve  func(sku string, quantity int, call int) error
	failRelease  func(sku string, quantity int, call int) error
}

func newMemoryOrderInventoryRepo() *memoryOrderInventoryRepo {
	return &memoryOrderInventoryRepo{items: make(map[string]domain.Inventory)}
}

func (r *memoryOrderInventoryRepo) Reserve(sku string, quantity int) (domain.Inventory, error) {
	r.reserveCalls++
	if r.failReserve != nil {
		if err := r.failReserve(sku, quantity, r.reserveCalls); err != nil {
			return domain.Inventory{}, err
		}
	}
	inv, ok := r.items[sku]
	if !ok {
		return domain.Inventory{}, errors.New("not found")
	}
	if err := inv.Reserve(quantity); err != nil {
		return domain.Inventory{}, err
	}
	r.items[sku] = inv
	return inv, nil
}

func (r *memoryOrderInventoryRepo) Release(sku string, quantity int) (domain.Inventory, error) {
	r.releaseCalls++
	if r.failRelease != nil {
		if err := r.failRelease(sku, quantity, r.releaseCalls); err != nil {
			return domain.Inventory{}, err
		}
	}
	inv, ok := r.items[sku]
	if !ok {
		return domain.Inventory{}, errors.New("not found")
	}
	if err := inv.Release(quantity); err != nil {
		return domain.Inventory{}, err
	}
	r.items[sku] = inv
	return inv, nil
}

func seedInventory(t *testing.T, inventories *memoryOrderInventoryRepo, sku string, onHand int, reserved int) {
	t.Helper()
	inv, err := domain.NewInventory(sku, onHand, reserved)
	if err != nil {
		t.Fatalf("failed to create inventory seed: %v", err)
	}
	inventories.items[sku] = inv
}

func newOrderUsecaseForTest(orders *memoryOrderRepo, payments *memoryPaymentRepo, inventories *memoryOrderInventoryRepo, id string) *OrderUsecase {
	return NewOrderUsecase(orders, payments, inventories, func() string { return id })
}

func TestCreateOrder_正常系_在庫確保と注文保存が成功する(t *testing.T) {
	orders := newMemoryOrderRepo()
	payments := newMemoryPaymentRepo()
	inventories := newMemoryOrderInventoryRepo()
	seedInventory(t, inventories, "sku-1", 10, 0)
	uc := newOrderUsecaseForTest(orders, payments, inventories, "order-1")

	out, err := uc.CreateOrder(CreateOrderInput{CustomerID: "c-1", Items: []domain.OrderItem{{SKU: "sku-1", Quantity: 2}}})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.OrderID != "order-1" {
		t.Fatalf("expected order-1, got %s", out.OrderID)
	}
	if out.Status != domain.OrderStatusAccepted {
		t.Fatalf("expected accepted, got %s", out.Status)
	}

	savedInv := inventories.items["sku-1"]
	if savedInv.OnHand != 10 || savedInv.Reserved != 2 || savedInv.Available() != 8 {
		t.Fatalf("expected inventory (10,2,8), got (%d,%d,%d)", savedInv.OnHand, savedInv.Reserved, savedInv.Available())
	}
}

func TestCreateOrder_異常系_在庫確保途中失敗時は副作用を残さない(t *testing.T) {
	orders := newMemoryOrderRepo()
	payments := newMemoryPaymentRepo()
	inventories := newMemoryOrderInventoryRepo()
	seedInventory(t, inventories, "sku-1", 10, 0)
	uc := newOrderUsecaseForTest(orders, payments, inventories, "order-1")

	_, err := uc.CreateOrder(CreateOrderInput{
		CustomerID: "c-1",
		Items: []domain.OrderItem{
			{SKU: "sku-1", Quantity: 1},
			{SKU: "missing", Quantity: 1},
		},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if orders.createCalls != 0 {
		t.Fatalf("create must not be called, got %d", orders.createCalls)
	}
	savedInv := inventories.items["sku-1"]
	if savedInv.OnHand != 10 || savedInv.Reserved != 0 || savedInv.Available() != 10 {
		t.Fatalf("expected inventory restored to (10,0,10), got (%d,%d,%d)", savedInv.OnHand, savedInv.Reserved, savedInv.Available())
	}
}

func TestCreateOrder_異常系_注文保存失敗時は在庫確保を補償する(t *testing.T) {
	orders := newMemoryOrderRepo()
	orders.failCreate = true
	payments := newMemoryPaymentRepo()
	inventories := newMemoryOrderInventoryRepo()
	seedInventory(t, inventories, "sku-1", 10, 0)
	uc := newOrderUsecaseForTest(orders, payments, inventories, "order-1")

	_, err := uc.CreateOrder(CreateOrderInput{CustomerID: "c-1", Items: []domain.OrderItem{{SKU: "sku-1", Quantity: 2}}})
	if err == nil {
		t.Fatalf("expected error")
	}
	if _, ok := orders.items["order-1"]; ok {
		t.Fatalf("order must not be saved")
	}
	savedInv := inventories.items["sku-1"]
	if savedInv.OnHand != 10 || savedInv.Reserved != 0 || savedInv.Available() != 10 {
		t.Fatalf("expected inventory restored to (10,0,10), got (%d,%d,%d)", savedInv.OnHand, savedInv.Reserved, savedInv.Available())
	}
}

func TestCreateOrder_異常系_注文保存失敗かつ補償失敗は補償失敗を優先する(t *testing.T) {
	orders := newMemoryOrderRepo()
	orders.failCreate = true
	payments := newMemoryPaymentRepo()
	inventories := newMemoryOrderInventoryRepo()
	seedInventory(t, inventories, "sku-1", 10, 0)
	inventories.failRelease = func(sku string, quantity int, call int) error {
		return errors.New("release failed")
	}
	uc := newOrderUsecaseForTest(orders, payments, inventories, "order-1")

	_, err := uc.CreateOrder(CreateOrderInput{CustomerID: "c-1", Items: []domain.OrderItem{{SKU: "sku-1", Quantity: 2}}})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "compensation failed") {
		t.Fatalf("expected compensation failed error, got %v", err)
	}
	savedInv := inventories.items["sku-1"]
	if savedInv.OnHand != 10 || savedInv.Reserved != 2 || savedInv.Available() != 8 {
		t.Fatalf("expected inventory to remain changed on compensation failure, got (%d,%d,%d)", savedInv.OnHand, savedInv.Reserved, savedInv.Available())
	}
}

func TestGetOrder_存在する場合(t *testing.T) {
	orders := newMemoryOrderRepo()
	payments := newMemoryPaymentRepo()
	inventories := newMemoryOrderInventoryRepo()
	uc := newOrderUsecaseForTest(orders, payments, inventories, "order-1")

	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 1}})
	_ = orders.Create(order)

	got, ok := uc.GetOrder("order-1")
	if !ok {
		t.Fatalf("expected order to exist")
	}
	if got.ID != "order-1" {
		t.Fatalf("expected order-1, got %s", got.ID)
	}
}

func TestCancelOrder_正常系_在庫を戻して注文を更新する(t *testing.T) {
	orders := newMemoryOrderRepo()
	payments := newMemoryPaymentRepo()
	inventories := newMemoryOrderInventoryRepo()
	seedInventory(t, inventories, "sku-1", 10, 2)
	uc := newOrderUsecaseForTest(orders, payments, inventories, "order-1")

	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 2}})
	_ = orders.Create(order)

	canceled, err := uc.CancelOrder("order-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if canceled.Status != domain.OrderStatusCanceled {
		t.Fatalf("expected canceled, got %s", canceled.Status)
	}
	savedInv := inventories.items["sku-1"]
	if savedInv.OnHand != 10 || savedInv.Reserved != 0 || savedInv.Available() != 10 {
		t.Fatalf("expected inventory released to (10,0,10), got (%d,%d,%d)", savedInv.OnHand, savedInv.Reserved, savedInv.Available())
	}
}

func TestCancelOrder_異常系_在庫戻し失敗時は注文を更新しない(t *testing.T) {
	orders := newMemoryOrderRepo()
	payments := newMemoryPaymentRepo()
	inventories := newMemoryOrderInventoryRepo()
	seedInventory(t, inventories, "sku-1", 10, 2)
	inventories.failRelease = func(sku string, quantity int, call int) error {
		return errors.New("release failed")
	}
	uc := newOrderUsecaseForTest(orders, payments, inventories, "order-1")

	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 2}})
	_ = orders.Create(order)

	_, err := uc.CancelOrder("order-1")
	if err == nil {
		t.Fatalf("expected error")
	}
	if orders.updateCalls != 0 {
		t.Fatalf("order update must not be called, got %d", orders.updateCalls)
	}
	savedInv := inventories.items["sku-1"]
	if savedInv.OnHand != 10 || savedInv.Reserved != 2 || savedInv.Available() != 8 {
		t.Fatalf("expected inventory unchanged (10,2,8), got (%d,%d,%d)", savedInv.OnHand, savedInv.Reserved, savedInv.Available())
	}
}

func TestCancelOrder_異常系_注文更新失敗時は在庫戻しを補償する(t *testing.T) {
	orders := newMemoryOrderRepo()
	orders.failUpdate = true
	payments := newMemoryPaymentRepo()
	inventories := newMemoryOrderInventoryRepo()
	seedInventory(t, inventories, "sku-1", 10, 2)
	uc := newOrderUsecaseForTest(orders, payments, inventories, "order-1")

	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 2}})
	_ = orders.Create(order)

	_, err := uc.CancelOrder("order-1")
	if err == nil {
		t.Fatalf("expected error")
	}
	savedInv := inventories.items["sku-1"]
	if savedInv.OnHand != 10 || savedInv.Reserved != 2 || savedInv.Available() != 8 {
		t.Fatalf("expected inventory restored by compensation to (10,2,8), got (%d,%d,%d)", savedInv.OnHand, savedInv.Reserved, savedInv.Available())
	}
}

func TestCancelOrder_異常系_注文更新失敗かつ補償失敗は補償失敗を優先する(t *testing.T) {
	orders := newMemoryOrderRepo()
	orders.failUpdate = true
	payments := newMemoryPaymentRepo()
	inventories := newMemoryOrderInventoryRepo()
	seedInventory(t, inventories, "sku-1", 10, 2)
	inventories.failReserve = func(sku string, quantity int, call int) error {
		return errors.New("reserve failed")
	}
	uc := newOrderUsecaseForTest(orders, payments, inventories, "order-1")

	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 2}})
	_ = orders.Create(order)

	_, err := uc.CancelOrder("order-1")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "compensation failed") {
		t.Fatalf("expected compensation failed error, got %v", err)
	}
	savedInv := inventories.items["sku-1"]
	if savedInv.OnHand != 10 || savedInv.Reserved != 0 || savedInv.Available() != 10 {
		t.Fatalf("expected release applied and compensation failed, got (%d,%d,%d)", savedInv.OnHand, savedInv.Reserved, savedInv.Available())
	}
}

func TestCancelOrder_存在しない場合(t *testing.T) {
	orders := newMemoryOrderRepo()
	payments := newMemoryPaymentRepo()
	inventories := newMemoryOrderInventoryRepo()
	uc := newOrderUsecaseForTest(orders, payments, inventories, "order-1")

	_, err := uc.CancelOrder("missing")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestCancelOrder_不変条件_既にcanceledの注文は失敗し在庫副作用が増えない(t *testing.T) {
	orders := newMemoryOrderRepo()
	payments := newMemoryPaymentRepo()
	inventories := newMemoryOrderInventoryRepo()
	seedInventory(t, inventories, "sku-1", 10, 0)
	uc := newOrderUsecaseForTest(orders, payments, inventories, "order-1")

	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 2}})
	order.Status = domain.OrderStatusCanceled
	_ = orders.Create(order)

	_, err := uc.CancelOrder("order-1")
	if err == nil {
		t.Fatalf("expected error")
	}
	if orders.updateCalls != 0 {
		t.Fatalf("order update side effect must not increase, got %d", orders.updateCalls)
	}
	if inventories.releaseCalls != 0 {
		t.Fatalf("inventory release side effect must not increase, got %d", inventories.releaseCalls)
	}
	savedInv := inventories.items["sku-1"]
	if savedInv.OnHand != 10 || savedInv.Reserved != 0 || savedInv.Available() != 10 {
		t.Fatalf("inventory must remain unchanged, got (%d,%d,%d)", savedInv.OnHand, savedInv.Reserved, savedInv.Available())
	}
}

func TestConfirmPayment_正常系(t *testing.T) {
	orders := newMemoryOrderRepo()
	payments := newMemoryPaymentRepo()
	inventories := newMemoryOrderInventoryRepo()
	uc := newOrderUsecaseForTest(orders, payments, inventories, "order-1")

	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 1}})
	_ = orders.Create(order)

	out, err := uc.ConfirmPayment(ConfirmPaymentInput{OrderID: "order-1", Amount: 100, IdempotencyKey: "k-1"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.PaymentStatus != "confirmed" {
		t.Fatalf("expected confirmed, got %s", out.PaymentStatus)
	}
}

func TestConfirmPayment_異常系_不正な入力(t *testing.T) {
	orders := newMemoryOrderRepo()
	payments := newMemoryPaymentRepo()
	inventories := newMemoryOrderInventoryRepo()
	uc := newOrderUsecaseForTest(orders, payments, inventories, "order-1")

	_, err := uc.ConfirmPayment(ConfirmPaymentInput{OrderID: "", Amount: 0, IdempotencyKey: ""})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, ErrConfirmPaymentInvalidRequest) {
		t.Fatalf("expected ErrConfirmPaymentInvalidRequest, got %v", err)
	}
}

func TestConfirmPayment_異常系_注文が存在しない(t *testing.T) {
	orders := newMemoryOrderRepo()
	payments := newMemoryPaymentRepo()
	inventories := newMemoryOrderInventoryRepo()
	uc := newOrderUsecaseForTest(orders, payments, inventories, "order-1")

	_, err := uc.ConfirmPayment(ConfirmPaymentInput{OrderID: "missing", Amount: 100, IdempotencyKey: "k-1"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, ErrConfirmPaymentNotFound) {
		t.Fatalf("expected ErrConfirmPaymentNotFound, got %v", err)
	}
}
