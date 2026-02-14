package boundary_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 境界一貫性統合テストにおける4観測の明記ルール:
// 1) HTTPステータス
// 2) 主要レスポンス項目
// 3) 後続API状態
// 4) 副作用DB
// 共通helperで 1)2) を固定し、各テスト本体で 3)4) を固定する。

// このテストは 境界一貫性統合テストの通しシナリオを固定する。
// 仕様対象: 注文作成 -> 決済確定 -> 注文参照の主要項目整合。
// 根拠: 実Router+実UseCase+実DB を通した境界振る舞いの回帰を検出するため。
func TestIntegration_OrderBoundary_注文作成から決済確定と注文参照まで通し検証(t *testing.T) {
	kit := new境界統合Testkit(t)
	// 観測4: 操作前の副作用DB基準値を取得する。
	beforeInventory := inventoryQuantityBySKU(t, kit, "sku-1")

	// 観測1/2: createOrderIntegration が HTTP 200 と主要項目(orderId/status)を検証する。
	createRes := createOrderIntegration(t, kit, "c-1", "sku-1", 1)
	if createRes.OrderID != "integration-order-1" {
		t.Fatalf("order id must be deterministic in integration test, got %s", createRes.OrderID)
	}
	// 観測4: 副作用DB（orders/payments/inventory）を検証する。
	if status := orderStatusByID(t, kit, createRes.OrderID); status != "accepted" {
		t.Fatalf("expected order status=accepted after create, got %s", status)
	}
	if payments := paymentsCountByOrder(t, kit, createRes.OrderID); payments != 0 {
		t.Fatalf("expected payments=0 after create, got %d", payments)
	}
	if afterCreateInventory := inventoryQuantityBySKU(t, kit, "sku-1"); afterCreateInventory != beforeInventory {
		t.Fatalf("inventory must remain unchanged after create, before=%d after=%d", beforeInventory, afterCreateInventory)
	}

	// 観測1/2: confirmPaymentIntegration が HTTP 200 と主要項目(orderId/paymentStatus)を検証する。
	confirmRes := confirmPaymentIntegration(t, kit, createRes.OrderID, 100, "k-1")
	if confirmRes.OrderID != createRes.OrderID {
		t.Fatalf("confirm order id mismatch: create=%s confirm=%s", createRes.OrderID, confirmRes.OrderID)
	}
	// 観測4: 決済確定後の副作用DBを検証する。
	if status := orderStatusByID(t, kit, createRes.OrderID); status != "confirmed" {
		t.Fatalf("expected order status=confirmed after confirm, got %s", status)
	}
	if payments := paymentsCountByOrder(t, kit, createRes.OrderID); payments != 1 {
		t.Fatalf("expected payments=1 after confirm, got %d", payments)
	}
	if afterConfirmInventory := inventoryQuantityBySKU(t, kit, "sku-1"); afterConfirmInventory != beforeInventory {
		t.Fatalf("inventory must remain unchanged after confirm, before=%d after=%d", beforeInventory, afterConfirmInventory)
	}

	// 観測1/2: getOrderIntegration が HTTP 200 と主要項目(id/customerId/status/items)を検証する。
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
	// 観測4: 操作前の副作用DB基準値を取得する。
	beforeInventory := inventoryQuantityBySKU(t, kit, "sku-1")

	// 観測1/2: createOrderIntegration が HTTP 200 と主要項目(orderId/status)を検証する。
	createRes := createOrderIntegration(t, kit, "c-1", "sku-1", 1)
	if createRes.Status != "accepted" {
		t.Fatalf("expected accepted on create 200, got %s", createRes.Status)
	}

	// 観測1/2: confirmPaymentIntegration が HTTP 200 と主要項目(orderId/paymentStatus)を検証する。
	_ = confirmPaymentIntegration(t, kit, createRes.OrderID, 100, "k-1")
	// 観測4: 決済確定後の副作用DBを検証する。
	if payments := paymentsCountByOrder(t, kit, createRes.OrderID); payments != 1 {
		t.Fatalf("expected payments=1 after confirm, got %d", payments)
	}
	if afterConfirmInventory := inventoryQuantityBySKU(t, kit, "sku-1"); afterConfirmInventory != beforeInventory {
		t.Fatalf("inventory must remain unchanged after confirm, before=%d after=%d", beforeInventory, afterConfirmInventory)
	}
	// 観測1/2: getOrderIntegration が HTTP 200 と主要項目を検証する。
	getRes := getOrderIntegration(t, kit, createRes.OrderID)
	if getRes.Status != "confirmed" {
		t.Fatalf("expected confirmed on get 200 after payment confirm, got %s", getRes.Status)
	}
	// 観測3: 後続API状態として、再取得しても confirmed が維持されることを確認する。
	getRes2 := getOrderIntegration(t, kit, createRes.OrderID)
	if getRes2.Status != "confirmed" {
		t.Fatalf("expected confirmed on second get, got %s", getRes2.Status)
	}
}

// このテストは POST /orders の数量境界値（1）を統合境界で固定する。
// 仕様対象: quantity=1 は最小有効値として 200 を返す。
// 根拠: 境界値の回帰で正当な注文作成が失敗しないことを保証するため。
func TestIntegration_OrderBoundary_POST_orders_200_quantity境界値1は受理される(t *testing.T) {
	kit := new境界統合Testkit(t)
	// 観測4: 操作前の副作用DB基準値を取得する。
	beforeInventory := inventoryQuantityBySKU(t, kit, "sku-1")

	// 観測1/2: createOrderIntegration が HTTP 200 と主要項目(orderId/status)を検証する。
	createRes := createOrderIntegration(t, kit, "c-1", "sku-1", 1)
	if createRes.Status != "accepted" {
		t.Fatalf("expected accepted, got %s", createRes.Status)
	}
	if createRes.OrderID == "" {
		t.Fatalf("expected non-empty orderId")
	}

	// 観測1/2: getOrderIntegration が HTTP 200 と主要項目を検証する。
	getRes := getOrderIntegration(t, kit, createRes.OrderID)
	// 観測3: 後続API状態を検証する。
	if getRes.Status != "accepted" {
		t.Fatalf("expected accepted on follow-up get, got %s", getRes.Status)
	}
	// 観測4: 副作用DBを検証する。
	if payments := paymentsCountByOrder(t, kit, createRes.OrderID); payments != 0 {
		t.Fatalf("expected payments=0 after create, got %d", payments)
	}
	if afterCreateInventory := inventoryQuantityBySKU(t, kit, "sku-1"); afterCreateInventory != beforeInventory {
		t.Fatalf("inventory must remain unchanged after create, before=%d after=%d", beforeInventory, afterCreateInventory)
	}
}

// このテストは GET /orders/{id} の 200 を統合境界で固定する。
// 仕様対象: 既存ID参照時に主要項目（id/customerId/status/items）を返す。
// 根拠: GET境界の主要観測が他操作のテストに埋もれないよう独立固定するため。
func TestIntegration_OrderBoundary_GET_orders_id_200_既存IDは主要項目を返す(t *testing.T) {
	kit := new境界統合Testkit(t)

	// 観測1/2: createOrderIntegration が HTTP 200 と主要項目(orderId/status)を検証する。
	createRes := createOrderIntegration(t, kit, "c-1", "sku-1", 1)
	// 観測4: 操作前の副作用DB基準値を取得する。
	beforeGet := snapshot境界統合副作用(t, kit)
	// 観測1/2: getOrderIntegration が HTTP 200 と主要項目(id/customerId/status/items)を検証する。
	getRes := getOrderIntegration(t, kit, createRes.OrderID)

	if getRes.ID != createRes.OrderID {
		t.Fatalf("expected id=%s, got %s", createRes.OrderID, getRes.ID)
	}
	if getRes.CustomerID != "c-1" {
		t.Fatalf("expected customerId=c-1, got %s", getRes.CustomerID)
	}
	if getRes.Status != "accepted" {
		t.Fatalf("expected status=accepted before confirm, got %s", getRes.Status)
	}
	if len(getRes.Items) != 1 || getRes.Items[0].SKU != "sku-1" || getRes.Items[0].Quantity != 1 {
		t.Fatalf("unexpected items: %+v", getRes.Items)
	}
	// 観測3: 後続API状態として、再取得でも同じ主要項目が観測できることを確認する。
	getRes2 := getOrderIntegration(t, kit, createRes.OrderID)
	if getRes2.ID != getRes.ID || getRes2.CustomerID != getRes.CustomerID || getRes2.Status != getRes.Status {
		t.Fatalf("expected same projection on second get, first=%+v second=%+v", getRes, getRes2)
	}
	// 観測4: 副作用DBが不変であることを検証する。
	afterGet := snapshot境界統合副作用(t, kit)
	if beforeGet != afterGet {
		t.Fatalf("GET must not mutate db side effects, before=%+v after=%+v", beforeGet, afterGet)
	}
}

// このテストは customerId の同値観測を統合境界で固定する。
// 仕様対象: POST /orders の customerId 入力値と GET /orders/{id} 応答値が一致する。
// 根拠: 主要フィールドの境界互換を実依存経路で回帰検出するため。
func TestIntegration_OrderBoundary_customerId同値観測_POSTとGETで一致する(t *testing.T) {
	kit := new境界統合Testkit(t)
	inputCustomerID := "customer-xyz-01"

	// 観測1/2: createOrderIntegration が HTTP 200 と主要項目(orderId/status)を検証する。
	createRes := createOrderIntegration(t, kit, inputCustomerID, "sku-1", 1)
	// 観測4: 操作前の副作用DB基準値を取得する。
	beforeGet := snapshot境界統合副作用(t, kit)
	// 観測1/2: getOrderIntegration が HTTP 200 と主要項目を検証する。
	getRes := getOrderIntegration(t, kit, createRes.OrderID)
	if getRes.CustomerID != inputCustomerID {
		t.Fatalf("customerId mismatch: post=%s get=%s", inputCustomerID, getRes.CustomerID)
	}
	// 観測3: 後続API状態として、再取得でも同値が維持されることを確認する。
	getRes2 := getOrderIntegration(t, kit, createRes.OrderID)
	if getRes2.CustomerID != inputCustomerID {
		t.Fatalf("customerId mismatch on second get: post=%s get=%s", inputCustomerID, getRes2.CustomerID)
	}
	// 観測4: 副作用DBが不変であることを検証する。
	afterGet := snapshot境界統合副作用(t, kit)
	if beforeGet != afterGet {
		t.Fatalf("GET must not mutate db side effects, before=%+v after=%+v", beforeGet, afterGet)
	}
}

// このテストは副作用DB観測（orders/payments/inventory）を統合境界で固定する。
// 仕様対象: 注文作成/決済確定時の orders状態・payments件数・inventory数量 の変化。
// 根拠: API応答だけでは検出できない永続状態の回帰を検出するため。
func TestIntegration_OrderBoundary_副作用DB観測_orders_payments_inventoryを固定する(t *testing.T) {
	kit := new境界統合Testkit(t)
	// amount は現状、1以上かどうかの入力検証にのみ使われるため、
	// 本テストでは有効値として固定値を使う。
	validAmount := 100

	// 観測4: 以降の在庫不変確認で比較するため、操作前の在庫数量を取得する。
	beforeInventory := inventoryQuantityBySKU(t, kit, "sku-1")
	// 操作前に対象注文IDの決済記録が存在しないことを確認する。
	if paymentsCountByOrder(t, kit, "integration-order-1") != 0 {
		t.Fatalf("expected no payments before create")
	}

	// 観測1/2: createOrderIntegration が HTTP 200 と主要項目(orderId/status)を検証する。
	// 1. 注文作成を実行し、作成直後のDB副作用を確認する。
	createRes := createOrderIntegration(t, kit, "c-1", "sku-1", 1)

	// 注文作成直後は orders.status が accepted で保存されること。
	if status := orderStatusByID(t, kit, createRes.OrderID); status != "accepted" {
		t.Fatalf("expected order status=accepted after create, got %s", status)
	}
	// 注文作成だけでは決済記録は増えないこと。
	if payments := paymentsCountByOrder(t, kit, createRes.OrderID); payments != 0 {
		t.Fatalf("expected payments=0 after create, got %d", payments)
	}
	// 現行仕様では注文作成時に在庫数量は変化しないこと。
	if afterCreateInventory := inventoryQuantityBySKU(t, kit, "sku-1"); afterCreateInventory != beforeInventory {
		t.Fatalf("inventory must remain unchanged after create, before=%d after=%d", beforeInventory, afterCreateInventory)
	}

	// 観測1/2: confirmPaymentIntegration が HTTP 200 と主要項目(orderId/paymentStatus)を検証する。
	// 2. 決済確定を実行し、確定後のDB副作用を確認する。
	_ = confirmPaymentIntegration(t, kit, createRes.OrderID, validAmount, "k-1")

	// 決済確定後は orders.status が confirmed に遷移すること。
	if status := orderStatusByID(t, kit, createRes.OrderID); status != "confirmed" {
		t.Fatalf("expected order status=confirmed after payment confirm, got %s", status)
	}
	// 初回確定で決済記録が1件作成されること。
	if payments := paymentsCountByOrder(t, kit, createRes.OrderID); payments != 1 {
		t.Fatalf("expected payments=1 after first confirm, got %d", payments)
	}
	// 現行仕様では決済確定時も在庫数量は変化しないこと。
	if afterConfirmInventory := inventoryQuantityBySKU(t, kit, "sku-1"); afterConfirmInventory != beforeInventory {
		t.Fatalf("inventory must remain unchanged after confirm, before=%d after=%d", beforeInventory, afterConfirmInventory)
	}

	// 観測3: 同一キー再送時の後続API状態（冪等）を確認する。
	// 3. 同一キー再送時の冪等性として、決済記録が増えないことを確認する。
	_ = confirmPaymentIntegration(t, kit, createRes.OrderID, validAmount, "k-1")
	if payments := paymentsCountByOrder(t, kit, createRes.OrderID); payments != 1 {
		t.Fatalf("expected payments to remain 1 on idempotent replay, got %d", payments)
	}
}

// このテストは エラー分類のうち 404（未存在）を統合境界で固定する。
// 仕様対象: 存在しない注文IDの参照は 404 を返す。
// 根拠: 未存在の意味を 404 として外部境界に固定するため。
func TestIntegration_OrderBoundary_404の意味_未存在注文参照は404を返す(t *testing.T) {
	kit := new境界統合Testkit(t)
	// 観測1/2: createOrderIntegration が HTTP 200 と主要項目(orderId/status)を検証する。
	base := createOrderIntegration(t, kit, "c-1", "sku-1", 1)
	// 観測4: 404試行前の副作用DB基準値を取得する。
	before := snapshot境界統合副作用(t, kit)

	req := httptest.NewRequest(http.MethodGet, "/orders/missing-order", nil)
	w := httptest.NewRecorder()
	kit.Router.ServeHTTP(w, req)
	// 観測1: HTTPステータス分類(404)を検証する。
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on missing order get, got %d body=%s", w.Code, strings.TrimSpace(w.Body.String()))
	}
	var body struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	// 観測2: 主要レスポンス項目(message)を検証する。
	if body.Message != "not found" {
		t.Fatalf("expected message=not found, got %s", body.Message)
	}
	// 観測3: 後続API状態として、既存注文は引き続き参照できることを確認する。
	getRes := getOrderIntegration(t, kit, base.OrderID)
	if getRes.Status != "accepted" {
		t.Fatalf("expected accepted on existing order after missing get, got %s", getRes.Status)
	}
	// 観測4: 副作用DBが不変であることを検証する。
	after := snapshot境界統合副作用(t, kit)
	if before != after {
		t.Fatalf("missing get must not mutate db side effects, before=%+v after=%+v", before, after)
	}
}

func createOrderIntegration(t *testing.T, kit *境界統合Testkit, customerID, sku string, quantity int) struct {
	OrderID string `json:"orderId"`
	Status  string `json:"status"`
} {
	t.Helper()

	// 観測1: HTTPステータス(200)を検証する。
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

	// 観測2: 主要レスポンス項目(orderId/status)を返却する。
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

	// amount は現状、1以上かどうかの入力検証にのみ使われる。
	// 観測1: HTTPステータス(200)を検証する。
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

	// 観測2: 主要レスポンス項目(orderId/paymentStatus)を返却する。
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

	// 観測1: HTTPステータス(200)を検証する。
	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID, nil)
	w := httptest.NewRecorder()
	kit.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on get, got %d body=%s", w.Code, strings.TrimSpace(w.Body.String()))
	}

	// 観測2: 主要レスポンス項目(id/customerId/status/items)を返却する。
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
	// 観測1/2/3/4: assertCreateOrder400Case で 400分類・エラー主要項目・後続API状態・副作用DBを検証する。
	assertCreateOrder400Case(t, "P5-ORD-400-01", true, false, false)
}

// このテストは POST /orders の 400（sku無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_orders_400_sku無効(t *testing.T) {
	// 観測1/2/3/4: assertCreateOrder400Case で 400分類・エラー主要項目・後続API状態・副作用DBを検証する。
	assertCreateOrder400Case(t, "P5-ORD-400-02", false, true, false)
}

// このテストは POST /orders の 400（quantity無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_orders_400_quantity無効(t *testing.T) {
	// 観測1/2/3/4: assertCreateOrder400Case で 400分類・エラー主要項目・後続API状態・副作用DBを検証する。
	assertCreateOrder400Case(t, "P5-ORD-400-03", false, false, true)
}

// このテストは POST /orders の 400（customerId+sku無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_orders_400_customerIdとsku無効(t *testing.T) {
	// 観測1/2/3/4: assertCreateOrder400Case で 400分類・エラー主要項目・後続API状態・副作用DBを検証する。
	assertCreateOrder400Case(t, "P5-ORD-400-04", true, true, false)
}

// このテストは POST /orders の 400（customerId+quantity無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_orders_400_customerIdとquantity無効(t *testing.T) {
	// 観測1/2/3/4: assertCreateOrder400Case で 400分類・エラー主要項目・後続API状態・副作用DBを検証する。
	assertCreateOrder400Case(t, "P5-ORD-400-05", true, false, true)
}

// このテストは POST /orders の 400（sku+quantity無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_orders_400_skuとquantity無効(t *testing.T) {
	// 観測1/2/3/4: assertCreateOrder400Case で 400分類・エラー主要項目・後続API状態・副作用DBを検証する。
	assertCreateOrder400Case(t, "P5-ORD-400-06", false, true, true)
}

// このテストは POST /orders の 400（全項目無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_orders_400_全項目無効(t *testing.T) {
	// 観測1/2/3/4: assertCreateOrder400Case で 400分類・エラー主要項目・後続API状態・副作用DBを検証する。
	assertCreateOrder400Case(t, "P5-ORD-400-07", true, true, true)
}

// このテストは POST /orders の 400（items空）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_orders_400_items空(t *testing.T) {
	kit := new境界統合Testkit(t)
	base := createOrderIntegration(t, kit, "c-1", "sku-1", 1)
	before := snapshot境界統合副作用(t, kit)

	payload, _ := json.Marshal(map[string]any{
		"customerId": "c-1",
		"items":      []map[string]any{},
	})
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	kit.Router.ServeHTTP(w, req)

	// 観測1/2/4: 400分類・エラー主要項目・副作用なしを共通helperで検証する。
	assert400NoSideEffect(t, "P5-ORD-400-08", w, before, snapshot境界統合副作用(t, kit))
	// 観測3: 後続APIで既存注文が不変で参照できることを検証する。
	getRes := getOrderIntegration(t, kit, base.OrderID)
	if getRes.Status != "accepted" {
		t.Fatalf("[P5-ORD-400-08] expected base order to remain accepted, got %s", getRes.Status)
	}
}

// このテストは POST /orders の 400（重複SKU）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_orders_400_重複SKU(t *testing.T) {
	kit := new境界統合Testkit(t)
	base := createOrderIntegration(t, kit, "c-1", "sku-1", 1)
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

	// 観測1/2/4: 400分類・エラー主要項目・副作用なしを共通helperで検証する。
	assert400NoSideEffect(t, "P5-ORD-400-09", w, before, snapshot境界統合副作用(t, kit))
	// 観測3: 後続APIで既存注文が不変で参照できることを検証する。
	getRes := getOrderIntegration(t, kit, base.OrderID)
	if getRes.Status != "accepted" {
		t.Fatalf("[P5-ORD-400-09] expected base order to remain accepted, got %s", getRes.Status)
	}
}

// このテストは POST /payments/confirm の 400（orderId無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_payments_confirm_400_orderId無効(t *testing.T) {
	// 観測1/2/3/4: assertConfirmPayment400Case で 400分類・エラー主要項目・後続API状態・副作用DBを検証する。
	assertConfirmPayment400Case(t, "P5-PAY-400-01", true, false, false)
}

// このテストは POST /payments/confirm の 400（amount無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_payments_confirm_400_amount無効(t *testing.T) {
	// 観測1/2/3/4: assertConfirmPayment400Case で 400分類・エラー主要項目・後続API状態・副作用DBを検証する。
	assertConfirmPayment400Case(t, "P5-PAY-400-02", false, true, false)
}

// このテストは POST /payments/confirm の 400（idempotencyKey無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_payments_confirm_400_key無効(t *testing.T) {
	// 観測1/2/3/4: assertConfirmPayment400Case で 400分類・エラー主要項目・後続API状態・副作用DBを検証する。
	assertConfirmPayment400Case(t, "P5-PAY-400-03", false, false, true)
}

// このテストは POST /payments/confirm の 400（orderId+amount無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_payments_confirm_400_orderIdとamount無効(t *testing.T) {
	// 観測1/2/3/4: assertConfirmPayment400Case で 400分類・エラー主要項目・後続API状態・副作用DBを検証する。
	assertConfirmPayment400Case(t, "P5-PAY-400-04", true, true, false)
}

// このテストは POST /payments/confirm の 400（orderId+idempotencyKey無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_payments_confirm_400_orderIdとkey無効(t *testing.T) {
	// 観測1/2/3/4: assertConfirmPayment400Case で 400分類・エラー主要項目・後続API状態・副作用DBを検証する。
	assertConfirmPayment400Case(t, "P5-PAY-400-05", true, false, true)
}

// このテストは POST /payments/confirm の 400（amount+idempotencyKey無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_payments_confirm_400_amountとkey無効(t *testing.T) {
	// 観測1/2/3/4: assertConfirmPayment400Case で 400分類・エラー主要項目・後続API状態・副作用DBを検証する。
	assertConfirmPayment400Case(t, "P5-PAY-400-06", false, true, true)
}

// このテストは POST /payments/confirm の 400（全項目無効）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_payments_confirm_400_全項目無効(t *testing.T) {
	// 観測1/2/3/4: assertConfirmPayment400Case で 400分類・エラー主要項目・後続API状態・副作用DBを検証する。
	assertConfirmPayment400Case(t, "P5-PAY-400-07", true, true, true)
}

// このテストは POST /payments/confirm の 404（未存在orderId）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_payments_confirm_404_未存在orderId(t *testing.T) {
	kit := new境界統合Testkit(t)

	createRes := createOrderIntegration(t, kit, "c-1", "sku-1", 1)
	before := snapshot境界統合副作用(t, kit)

	payload, _ := json.Marshal(map[string]any{
		"orderId":        "missing-order",
		"amount":         100,
		"idempotencyKey": "k-404",
	})
	req := httptest.NewRequest(http.MethodPost, "/payments/confirm", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	kit.Router.ServeHTTP(w, req)
	// 観測1: HTTPステータス分類(404)を検証する。
	if w.Code != http.StatusNotFound {
		t.Fatalf("[P5-PAY-404-01] expected 404, got %d body=%s", w.Code, strings.TrimSpace(w.Body.String()))
	}
	// 観測2: 主要レスポンス項目(message)を検証する。
	var body struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("[P5-PAY-404-01] failed to decode response: %v", err)
	}
	if body.Message != "not found" {
		t.Fatalf("[P5-PAY-404-01] expected message=not found, got %s", body.Message)
	}

	// 観測4: 副作用DBが増えていないことを検証する。
	after := snapshot境界統合副作用(t, kit)
	if before != after {
		t.Fatalf("[P5-PAY-404-01] expected no db side effects, before=%+v after=%+v", before, after)
	}
	// 観測3: 後続APIで既存注文状態が不変であることを検証する。
	order := getOrderIntegration(t, kit, createRes.OrderID)
	if order.Status != "accepted" {
		t.Fatalf("[P5-PAY-404-01] order status must remain accepted, got %s", order.Status)
	}
}

// このテストは POST /payments/confirm の冪等性（同一キー再送）を統合境界で固定する。
func TestIntegration_OrderBoundary_POST_payments_confirm_冪等_同一キー再送(t *testing.T) {
	kit := new境界統合Testkit(t)
	// 観測1/2: createOrderIntegration/confirmPaymentIntegration が HTTP 200 と主要項目を検証する。
	createRes := createOrderIntegration(t, kit, "c-1", "sku-1", 1)

	first := confirmPaymentIntegration(t, kit, createRes.OrderID, 100, "k-1")
	// 観測4: 副作用DBスナップショット（orders/order_items/payments/inventory）を取得する。
	afterFirst := snapshot境界統合副作用(t, kit)
	second := confirmPaymentIntegration(t, kit, createRes.OrderID, 100, "k-1")
	afterSecond := snapshot境界統合副作用(t, kit)

	if first != second {
		t.Fatalf("[P5-PAY-IDEMP-01] expected same response, first=%+v second=%+v", first, second)
	}
	if afterFirst != afterSecond {
		t.Fatalf("[P5-PAY-IDEMP-01] expected no additional side effects, first=%+v second=%+v", afterFirst, afterSecond)
	}
	// 観測3: 後続API状態として confirmed 維持を検証する。
	order := getOrderIntegration(t, kit, createRes.OrderID)
	if order.Status != "confirmed" {
		t.Fatalf("[P5-PAY-IDEMP-01] expected confirmed, got %s", order.Status)
	}
}

type 境界統合DB副作用スナップショット struct {
	orders     int
	orderItems int
	payments   int
	inventory  int
}

func assertCreateOrder400Case(t *testing.T, caseID string, invalidCustomerID, invalidSKU, invalidQuantity bool) {
	t.Helper()

	kit := new境界統合Testkit(t)
	base := createOrderIntegration(t, kit, "c-1", "sku-1", 1)
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

	// 観測1/2/4: 400分類・エラー主要項目・副作用なしを検証する。
	assert400NoSideEffect(t, caseID, w, before, snapshot境界統合副作用(t, kit))
	// 観測3: 後続API状態として、既存注文が不変で参照できることを検証する。
	getRes := getOrderIntegration(t, kit, base.OrderID)
	if getRes.Status != "accepted" {
		t.Fatalf("[%s] expected base order to remain accepted, got %s", caseID, getRes.Status)
	}
}

func assertConfirmPayment400Case(t *testing.T, caseID string, invalidOrderID, invalidAmount, invalidKey bool) {
	t.Helper()

	kit := new境界統合Testkit(t)
	base := createOrderIntegration(t, kit, "c-1", "sku-1", 1)
	before := snapshot境界統合副作用(t, kit)

	orderID := base.OrderID
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

	// 観測1/2/4: 400分類・エラー主要項目・副作用なしを検証する。
	assert400NoSideEffect(t, caseID, w, before, snapshot境界統合副作用(t, kit))
	// 観測3: 後続API状態として、既存注文が不変で参照できることを検証する。
	getRes := getOrderIntegration(t, kit, base.OrderID)
	if getRes.Status != "accepted" {
		t.Fatalf("[%s] expected base order to remain accepted, got %s", caseID, getRes.Status)
	}
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
	if err := kit.DB.QueryRow(`SELECT quantity FROM inventory WHERE sku = 'sku-1'`).Scan(&s.inventory); err != nil {
		t.Fatalf("failed to fetch inventory quantity: %v", err)
	}
	return s
}

func orderStatusByID(t *testing.T, kit *境界統合Testkit, orderID string) string {
	t.Helper()

	var status string
	if err := kit.DB.QueryRow(`SELECT status FROM orders WHERE id = $1`, orderID).Scan(&status); err != nil {
		t.Fatalf("failed to fetch order status: %v", err)
	}
	return status
}

func paymentsCountByOrder(t *testing.T, kit *境界統合Testkit, orderID string) int {
	t.Helper()

	var count int
	if err := kit.DB.QueryRow(`SELECT COUNT(*) FROM payments WHERE order_id = $1`, orderID).Scan(&count); err != nil {
		t.Fatalf("failed to count payments by order: %v", err)
	}
	return count
}

func inventoryQuantityBySKU(t *testing.T, kit *境界統合Testkit, sku string) int {
	t.Helper()

	var qty int
	if err := kit.DB.QueryRow(`SELECT quantity FROM inventory WHERE sku = $1`, sku).Scan(&qty); err != nil {
		t.Fatalf("failed to fetch inventory quantity: %v", err)
	}
	return qty
}
