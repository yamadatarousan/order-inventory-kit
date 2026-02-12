package boundary_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// このテストは 境界一貫性統合テストの通しシナリオを固定する。
// 仕様対象: 注文作成 -> 決済確定 -> 注文参照の主要項目整合。
// 根拠: 実Router+実UseCase+実DB を通した境界振る舞いの回帰を検出するため。
func TestIntegration_OrderBoundary_注文作成から決済確定と注文参照まで通し検証(t *testing.T) {
	kit := new境界統合Testkit(t)

	createRes := createOrderIntegration(t, kit, "c-1", "sku-1", 1)
	if createRes.OrderID != "integration-order-1" {
		t.Fatalf("order id must be deterministic in integration test, got %s", createRes.OrderID)
	}
	confirmRes := confirmPaymentIntegration(t, kit, createRes.OrderID, 100, "k-1")
	if confirmRes.OrderID != createRes.OrderID {
		t.Fatalf("confirm order id mismatch: create=%s confirm=%s", createRes.OrderID, confirmRes.OrderID)
	}
	getRes := getOrderIntegration(t, kit, createRes.OrderID)
	if getRes.ID != createRes.OrderID {
		t.Fatalf("get id mismatch: create=%s get=%s", createRes.OrderID, getRes.ID)
	}
	if getRes.CustomerID != "c-1" {
		t.Fatalf("expected customerId=c-1, got %s", getRes.CustomerID)
	}
	if getRes.Status != "confirmed" {
		t.Fatalf("expected status=confirmed after payment confirm, got %s", getRes.Status)
	}
	if len(getRes.Items) != 1 || getRes.Items[0].SKU != "sku-1" || getRes.Items[0].Quantity != 1 {
		t.Fatalf("unexpected items: %+v", getRes.Items)
	}
}

// このテストは 200 の意味（accepted -> confirmed）を統合境界で固定する。
// 仕様対象: 作成200時は accepted、決済確定200後の参照200時は confirmed。
// 根拠: 同じ 200 でも操作ごとの意味差を回帰で崩さないようにするため。
func TestIntegration_OrderBoundary_200の意味_acceptedからconfirmedへの遷移を固定する(t *testing.T) {
	kit := new境界統合Testkit(t)

	createRes := createOrderIntegration(t, kit, "c-1", "sku-1", 1)
	if createRes.Status != "accepted" {
		t.Fatalf("expected accepted on create 200, got %s", createRes.Status)
	}

	_ = confirmPaymentIntegration(t, kit, createRes.OrderID, 100, "k-1")
	getRes := getOrderIntegration(t, kit, createRes.OrderID)
	if getRes.Status != "confirmed" {
		t.Fatalf("expected confirmed on get 200 after payment confirm, got %s", getRes.Status)
	}
}

// このテストは POST /orders の数量境界値（1）を統合境界で固定する。
// 仕様対象: quantity=1 は最小有効値として 200 を返す。
// 根拠: 境界値の回帰で正当な注文作成が失敗しないことを保証するため。
func TestIntegration_OrderBoundary_POST_orders_200_quantity境界値1は受理される(t *testing.T) {
	kit := new境界統合Testkit(t)
	res := createOrderIntegration(t, kit, "c-1", "sku-1", 1)
	if res.Status != "accepted" {
		t.Fatalf("expected accepted, got %s", res.Status)
	}
}

// このテストは エラー分類のうち 404（未存在）を統合境界で固定する。
// 仕様対象: 存在しない注文IDの参照は 404 を返す。
// 根拠: 未存在の意味を 404 として外部境界に固定するため。
func TestIntegration_OrderBoundary_404の意味_未存在注文参照は404を返す(t *testing.T) {
	kit := new境界統合Testkit(t)

	req := httptest.NewRequest(http.MethodGet, "/orders/missing-order", nil)
	w := httptest.NewRecorder()
	kit.Router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on missing order get, got %d body=%s", w.Code, strings.TrimSpace(w.Body.String()))
	}
}

func createOrderIntegration(t *testing.T, kit *境界統合Testkit, customerID, sku string, quantity int) struct {
	OrderID string `json:"orderId"`
	Status  string `json:"status"`
} {
	t.Helper()

	payload, _ := json.Marshal(map[string]any{
		"customerId": customerID,
		"items":      []map[string]any{{"sku": sku, "quantity": quantity}},
	})
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	kit.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on create, got %d body=%s", w.Code, strings.TrimSpace(w.Body.String()))
	}

	var res struct {
		OrderID string `json:"orderId"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	return res
}

func confirmPaymentIntegration(t *testing.T, kit *境界統合Testkit, orderID string, amount int, key string) struct {
	OrderID       string `json:"orderId"`
	PaymentStatus string `json:"paymentStatus"`
} {
	t.Helper()

	payload, _ := json.Marshal(map[string]any{
		"orderId":        orderID,
		"amount":         amount,
		"idempotencyKey": key,
	})
	req := httptest.NewRequest(http.MethodPost, "/payments/confirm", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	kit.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on confirm, got %d body=%s", w.Code, strings.TrimSpace(w.Body.String()))
	}

	var res struct {
		OrderID       string `json:"orderId"`
		PaymentStatus string `json:"paymentStatus"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode confirm response: %v", err)
	}
	return res
}

func getOrderIntegration(t *testing.T, kit *境界統合Testkit, orderID string) struct {
	ID         string `json:"id"`
	CustomerID string `json:"customerId"`
	Status     string `json:"status"`
	Items      []struct {
		SKU      string `json:"sku"`
		Quantity int    `json:"quantity"`
	} `json:"items"`
} {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID, nil)
	w := httptest.NewRecorder()
	kit.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on get, got %d body=%s", w.Code, strings.TrimSpace(w.Body.String()))
	}

	var res struct {
		ID         string `json:"id"`
		CustomerID string `json:"customerId"`
		Status     string `json:"status"`
		Items      []struct {
			SKU      string `json:"sku"`
			Quantity int    `json:"quantity"`
		} `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode get response: %v", err)
	}
	return res
}

// このテストは POST /orders の 400（customerId無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_orders_400_customerId無効(t *testing.T) {
	assertCreateOrder400Case(t, "P5-ORD-400-01", true, false, false)
}

// このテストは POST /orders の 400（sku無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_orders_400_sku無効(t *testing.T) {
	assertCreateOrder400Case(t, "P5-ORD-400-02", false, true, false)
}

// このテストは POST /orders の 400（quantity無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_orders_400_quantity無効(t *testing.T) {
	assertCreateOrder400Case(t, "P5-ORD-400-03", false, false, true)
}

// このテストは POST /orders の 400（customerId+sku無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_orders_400_customerIdとsku無効(t *testing.T) {
	assertCreateOrder400Case(t, "P5-ORD-400-04", true, true, false)
}

// このテストは POST /orders の 400（customerId+quantity無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_orders_400_customerIdとquantity無効(t *testing.T) {
	assertCreateOrder400Case(t, "P5-ORD-400-05", true, false, true)
}

// このテストは POST /orders の 400（sku+quantity無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_orders_400_skuとquantity無効(t *testing.T) {
	assertCreateOrder400Case(t, "P5-ORD-400-06", false, true, true)
}

// このテストは POST /orders の 400（全項目無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_orders_400_全項目無効(t *testing.T) {
	assertCreateOrder400Case(t, "P5-ORD-400-07", true, true, true)
}

// このテストは POST /orders の 400（items空）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_orders_400_items空(t *testing.T) {
	kit := new境界統合Testkit(t)
	before := snapshot境界統合副作用(t, kit)

	payload, _ := json.Marshal(map[string]any{
		"customerId": "c-1",
		"items":      []map[string]any{},
	})
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	kit.Router.ServeHTTP(w, req)

	assert400NoSideEffect(t, "P5-ORD-400-08", w, before, snapshot境界統合副作用(t, kit))
}

// このテストは POST /orders の 400（重複SKU）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_orders_400_重複SKU(t *testing.T) {
	kit := new境界統合Testkit(t)
	before := snapshot境界統合副作用(t, kit)

	payload, _ := json.Marshal(map[string]any{
		"customerId": "c-1",
		"items": []map[string]any{
			{"sku": "sku-1", "quantity": 1},
			{"sku": "sku-1", "quantity": 2},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	kit.Router.ServeHTTP(w, req)

	assert400NoSideEffect(t, "P5-ORD-400-09", w, before, snapshot境界統合副作用(t, kit))
}

// このテストは POST /payments/confirm の 400（orderId無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_payments_confirm_400_orderId無効(t *testing.T) {
	assertConfirmPayment400Case(t, "P5-PAY-400-01", true, false, false)
}

// このテストは POST /payments/confirm の 400（amount無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_payments_confirm_400_amount無効(t *testing.T) {
	assertConfirmPayment400Case(t, "P5-PAY-400-02", false, true, false)
}

// このテストは POST /payments/confirm の 400（idempotencyKey無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_payments_confirm_400_key無効(t *testing.T) {
	assertConfirmPayment400Case(t, "P5-PAY-400-03", false, false, true)
}

// このテストは POST /payments/confirm の 400（orderId+amount無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_payments_confirm_400_orderIdとamount無効(t *testing.T) {
	assertConfirmPayment400Case(t, "P5-PAY-400-04", true, true, false)
}

// このテストは POST /payments/confirm の 400（orderId+idempotencyKey無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_payments_confirm_400_orderIdとkey無効(t *testing.T) {
	assertConfirmPayment400Case(t, "P5-PAY-400-05", true, false, true)
}

// このテストは POST /payments/confirm の 400（amount+idempotencyKey無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_payments_confirm_400_amountとkey無効(t *testing.T) {
	assertConfirmPayment400Case(t, "P5-PAY-400-06", false, true, true)
}

// このテストは POST /payments/confirm の 400（全項目無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_payments_confirm_400_全項目無効(t *testing.T) {
	assertConfirmPayment400Case(t, "P5-PAY-400-07", true, true, true)
}

// このテストはケース一覧を固定するためのスケルトン。
func TestIntegration_OrderBoundary_POST_payments_confirm_404_未存在orderId(t *testing.T) {
	未実装境界統合ケース(t, "P5-PAY-404-01")
}

// このテストはケース一覧を固定するためのスケルトン。
func TestIntegration_OrderBoundary_POST_payments_confirm_冪等_同一キー再送(t *testing.T) {
	未実装境界統合ケース(t, "P5-PAY-IDEMP-01")
}

func 未実装境界統合ケース(t *testing.T, caseID string) {
	t.Helper()
	t.Skipf("TODO: implement integration case %s", caseID)
}

type 境界統合DB副作用スナップショット struct {
	orders     int
	orderItems int
	payments   int
}

func assertCreateOrder400Case(t *testing.T, caseID string, invalidCustomerID, invalidSKU, invalidQuantity bool) {
	t.Helper()

	kit := new境界統合Testkit(t)
	before := snapshot境界統合副作用(t, kit)

	customerID := "c-1"
	if invalidCustomerID {
		customerID = ""
	}
	sku := "sku-1"
	if invalidSKU {
		sku = ""
	}
	quantity := 1
	if invalidQuantity {
		quantity = 0
	}

	payload, _ := json.Marshal(map[string]any{
		"customerId": customerID,
		"items":      []map[string]any{{"sku": sku, "quantity": quantity}},
	})
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	kit.Router.ServeHTTP(w, req)

	assert400NoSideEffect(t, caseID, w, before, snapshot境界統合副作用(t, kit))
}

func assertConfirmPayment400Case(t *testing.T, caseID string, invalidOrderID, invalidAmount, invalidKey bool) {
	t.Helper()

	kit := new境界統合Testkit(t)
	before := snapshot境界統合副作用(t, kit)

	orderID := "integration-order-1"
	if invalidOrderID {
		orderID = ""
	}
	amount := 100
	if invalidAmount {
		amount = 0
	}
	key := "k-1"
	if invalidKey {
		key = ""
	}

	payload, _ := json.Marshal(map[string]any{
		"orderId":        orderID,
		"amount":         amount,
		"idempotencyKey": key,
	})
	req := httptest.NewRequest(http.MethodPost, "/payments/confirm", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	kit.Router.ServeHTTP(w, req)

	assert400NoSideEffect(t, caseID, w, before, snapshot境界統合副作用(t, kit))
}

func assert400NoSideEffect(t *testing.T, caseID string, w *httptest.ResponseRecorder, before, after 境界統合DB副作用スナップショット) {
	t.Helper()

	if w.Code != http.StatusBadRequest {
		t.Fatalf("[%s] expected 400, got %d body=%s", caseID, w.Code, strings.TrimSpace(w.Body.String()))
	}
	var body struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("[%s] failed to decode error response: %v", caseID, err)
	}
	if body.Message != "invalid request" {
		t.Fatalf("[%s] expected message=invalid request, got %s", caseID, body.Message)
	}
	if before != after {
		t.Fatalf("[%s] expected no db side effects, before=%+v after=%+v", caseID, before, after)
	}
}

func snapshot境界統合副作用(t *testing.T, kit *境界統合Testkit) 境界統合DB副作用スナップショット {
	t.Helper()

	var s 境界統合DB副作用スナップショット
	if err := kit.DB.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&s.orders); err != nil {
		t.Fatalf("failed to count orders: %v", err)
	}
	if err := kit.DB.QueryRow(`SELECT COUNT(*) FROM order_items`).Scan(&s.orderItems); err != nil {
		t.Fatalf("failed to count order_items: %v", err)
	}
	if err := kit.DB.QueryRow(`SELECT COUNT(*) FROM payments`).Scan(&s.payments); err != nil {
		t.Fatalf("failed to count payments: %v", err)
	}
	return s
}
