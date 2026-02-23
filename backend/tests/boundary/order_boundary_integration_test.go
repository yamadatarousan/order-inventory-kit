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
	beforeInventory := inventoryStateBySKU(t, kit, "sku-1")

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
	afterCreateInventory := inventoryStateBySKU(t, kit, "sku-1")
	assertInventoryReserveTransition(t, "after create", beforeInventory, afterCreateInventory, 1)

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
	afterConfirmInventory := inventoryStateBySKU(t, kit, "sku-1")
	assertInventoryUnchanged(t, "after confirm", afterCreateInventory, afterConfirmInventory)

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
	beforeInventory := inventoryStateBySKU(t, kit, "sku-1")

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
	afterConfirmInventory := inventoryStateBySKU(t, kit, "sku-1")
	assertInventoryReserveTransition(t, "after create+confirm", beforeInventory, afterConfirmInventory, 1)
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
	beforeInventory := inventoryStateBySKU(t, kit, "sku-1")

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
	afterCreateInventory := inventoryStateBySKU(t, kit, "sku-1")
	assertInventoryReserveTransition(t, "after create", beforeInventory, afterCreateInventory, 1)
}

// このテストは 注文作成と在庫引当の整合方針（補償処理）を統合境界で固定する。
// 仕様対象: 複数明細の在庫確保途中で失敗した場合、先行確保分を補償で戻し副作用を残さない。
// 根拠: 同一トランザクション非採用時でも、外部観測として一貫した失敗結果を保証するため。
func TestIntegration_OrderBoundary_POST_orders_在庫確保途中失敗は補償で副作用を残さない(t *testing.T) {
	kit := new境界統合Testkit(t)
	// 観測4: 失敗試行前の副作用DB基準値を取得する。
	before := snapshot境界統合副作用(t, kit)
	beforeInventory := inventoryStateBySKU(t, kit, "sku-1")

	payload, _ := json.Marshal(map[string]any{
		"customerId": "c-1",
		"items": []map[string]any{
			{"sku": "sku-1", "quantity": 1},
			{"sku": "missing-sku", "quantity": 1},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	kit.Router.ServeHTTP(w, req)

	// 観測1/2/4: 400分類・エラー主要項目・副作用なし（補償で基準値に戻る）を検証する。
	assert400NoSideEffect(t, "P5-ORD-COMP-01", w, before, snapshot境界統合副作用(t, kit))
	// 観測3: 後続API状態として、次の正常注文が成功し在庫遷移も一貫することを検証する。
	createRes := createOrderIntegration(t, kit, "c-1", "sku-1", 1)
	if createRes.Status != "accepted" {
		t.Fatalf("[P5-ORD-COMP-01] expected accepted after compensation path, got %s", createRes.Status)
	}
	afterCreateInventory := inventoryStateBySKU(t, kit, "sku-1")
	assertInventoryReserveTransition(t, "P5-ORD-COMP-01 after recovery create", beforeInventory, afterCreateInventory, 1)
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
// 仕様対象: 注文作成/決済確定時の orders状態・payments件数・inventory(on_hand/reserved/available) の変化。
// 根拠: API応答だけでは検出できない永続状態の回帰を検出するため。
func TestIntegration_OrderBoundary_副作用DB観測_orders_payments_inventoryを固定する(t *testing.T) {
	kit := new境界統合Testkit(t)
	// amount は注文合計との照合に使われるため、seed済み単価100・数量1の合計で固定する。
	validAmount := 100

	// 観測4: 以降の在庫遷移確認で比較するため、操作前の在庫状態を取得する。
	beforeInventory := inventoryStateBySKU(t, kit, "sku-1")
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
	// 標準在庫モデルでは注文作成時に引当が発生し、on_hand不変・reserved増・available減となること。
	afterCreateInventory := inventoryStateBySKU(t, kit, "sku-1")
	assertInventoryReserveTransition(t, "after create", beforeInventory, afterCreateInventory, 1)

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
	// 標準在庫モデルでは決済確定で在庫は減算せず、注文作成時点の引当状態を維持すること。
	afterConfirmInventory := inventoryStateBySKU(t, kit, "sku-1")
	assertInventoryUnchanged(t, "after confirm", afterCreateInventory, afterConfirmInventory)

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
	// 観測1/2/4: 404分類・エラー主要項目・副作用なしを共通helperで検証する。
	assert404NoSideEffect(t, "P5-ORD-404-01", w, before, snapshot境界統合副作用(t, kit))
	// 観測3: 後続API状態として、既存注文は引き続き参照できることを確認する。
	getRes := getOrderIntegration(t, kit, base.OrderID)
	if getRes.Status != "accepted" {
		t.Fatalf("expected accepted on existing order after missing get, got %s", getRes.Status)
	}
}

// createOrderIntegration は POST /orders の成功系呼び出しを共通化する。
// 4観測のうち HTTPステータス(200) と主要レスポンス項目(orderId/status) をこの関数で固定する。
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

// confirmPaymentIntegration は POST /payments/confirm の成功系呼び出しを共通化する。
// 4観測のうち HTTPステータス(200) と主要レスポンス項目(orderId/paymentStatus) をこの関数で固定する。
func confirmPaymentIntegration(t *testing.T, kit *境界統合Testkit, orderID string, amount int, key string) struct {
	OrderID       string `json:"orderId"`
	PaymentStatus string `json:"paymentStatus"`
} {
	t.Helper()

	// amount は注文合計照合に使われるため、呼び出し側で妥当値/不一致値を使い分ける。
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

// getOrderIntegration は GET /orders/{id} の成功系呼び出しを共通化する。
// 4観測のうち HTTPステータス(200) と主要レスポンス項目(id/customerId/status/items) をこの関数で固定する。
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
	// 観測1/2/4: 404分類・エラー主要項目・副作用なしを共通helperで検証する。
	assert404NoSideEffect(t, "P5-PAY-404-01", w, before, snapshot境界統合副作用(t, kit))
	// 観測3: 後続APIで既存注文状態が不変であることを検証する。
	order := getOrderIntegration(t, kit, createRes.OrderID)
	if order.Status != "accepted" {
		t.Fatalf("[P5-PAY-404-01] order status must remain accepted, got %s", order.Status)
	}
}

// このテストは POST /payments/confirm の金額一致時200を統合境界で固定する。
// 仕様対象: amount が注文合計と一致する場合は 200 で確定し、後続参照は confirmed を返す。
// 根拠: 金額整合の正常系を境界で固定するため。
func TestIntegration_OrderBoundary_POST_payments_confirm_200_金額一致(t *testing.T) {
	kit := new境界統合Testkit(t)
	createRes := createOrderIntegration(t, kit, "c-1", "sku-1", 1)
	beforeConfirmInventory := inventoryStateBySKU(t, kit, "sku-1")

	// 観測1/2: confirmPaymentIntegration が HTTP 200 と主要項目を検証する。
	res := confirmPaymentIntegration(t, kit, createRes.OrderID, 100, "k-200")
	if res.PaymentStatus != "confirmed" {
		t.Fatalf("[P5-PAY-200-02] expected paymentStatus=confirmed, got %s", res.PaymentStatus)
	}
	// 観測3: 後続API状態として confirmed へ遷移していることを検証する。
	order := getOrderIntegration(t, kit, createRes.OrderID)
	if order.Status != "confirmed" {
		t.Fatalf("[P5-PAY-200-02] expected confirmed, got %s", order.Status)
	}
	// 観測4: 副作用DBとして決済件数更新と在庫不変を検証する。
	if payments := paymentsCountByOrder(t, kit, createRes.OrderID); payments != 1 {
		t.Fatalf("[P5-PAY-200-02] expected payments=1, got %d", payments)
	}
	afterConfirmInventory := inventoryStateBySKU(t, kit, "sku-1")
	assertInventoryUnchanged(t, "P5-PAY-200-02 after confirm", beforeConfirmInventory, afterConfirmInventory)
}

// このテストは POST /payments/confirm の 409（金額不一致）を統合境界で固定する。
// 仕様対象: amount が注文合計と不一致のとき 409 を返し、注文状態とDB副作用を変化させない。
// 根拠: 金額不一致を 400/404 ではなく 409 で区別するため。
func TestIntegration_OrderBoundary_POST_payments_confirm_409_金額不一致(t *testing.T) {
	kit := new境界統合Testkit(t)
	createRes := createOrderIntegration(t, kit, "c-1", "sku-1", 1)
	before := snapshot境界統合副作用(t, kit)

	payload, _ := json.Marshal(map[string]any{
		"orderId":        createRes.OrderID,
		"amount":         101,
		"idempotencyKey": "k-409-mismatch",
	})
	req := httptest.NewRequest(http.MethodPost, "/payments/confirm", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	kit.Router.ServeHTTP(w, req)

	// 観測1/2/4: 409分類・エラー主要項目・副作用なしを共通helperで検証する。
	assert409NoSideEffect(t, "P5-PAY-409-01", w, before, snapshot境界統合副作用(t, kit))
	// 観測3: 後続API状態として注文状態が accepted のまま維持されることを検証する。
	order := getOrderIntegration(t, kit, createRes.OrderID)
	if order.Status != "accepted" {
		t.Fatalf("[P5-PAY-409-01] expected accepted after amount mismatch, got %s", order.Status)
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

// このテストは POST /payments/confirm の冪等異額再送を統合境界で固定する。
// 仕様対象: 同一 idempotencyKey を異なる amount で再送した場合は 409 を返し、副作用を増やさない。
// 根拠: 冪等再送時の同額/異額の分類を固定するため。
func TestIntegration_OrderBoundary_POST_payments_confirm_冪等_異額再送は409(t *testing.T) {
	kit := new境界統合Testkit(t)
	createRes := createOrderIntegration(t, kit, "c-1", "sku-1", 1)
	_ = confirmPaymentIntegration(t, kit, createRes.OrderID, 100, "k-same-key")
	beforeSecond := snapshot境界統合副作用(t, kit)

	payload, _ := json.Marshal(map[string]any{
		"orderId":        createRes.OrderID,
		"amount":         101,
		"idempotencyKey": "k-same-key",
	})
	req := httptest.NewRequest(http.MethodPost, "/payments/confirm", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	kit.Router.ServeHTTP(w, req)

	// 観測1/2/4: 409分類・エラー主要項目・副作用なしを共通helperで検証する。
	assert409NoSideEffect(t, "P5-PAY-IDEMP-02", w, beforeSecond, snapshot境界統合副作用(t, kit))
	// 観測3: 後続API状態として confirmed が維持されることを検証する。
	order := getOrderIntegration(t, kit, createRes.OrderID)
	if order.Status != "confirmed" {
		t.Fatalf("[P5-PAY-IDEMP-02] expected confirmed, got %s", order.Status)
	}
}

type 境界統合DB副作用スナップショット struct {
	orders     int
	orderItems int
	payments   int
	inventory  inventoryState
}

// assertCreateOrder400Case は POST /orders の入力無効組み合わせを共通検証する。
// 400分類、エラーメッセージ、副作用なし、既存注文の後続状態不変を一括で固定する。
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

// assertConfirmPayment400Case は POST /payments/confirm の入力無効組み合わせを共通検証する。
// 400分類、エラーメッセージ、副作用なし、既存注文の後続状態不変を一括で固定する。
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

// assert400NoSideEffect は 400エラー時の共通契約を検証する。
// invalid request の返却と、副作用DBスナップショット不変を固定する。
func assert400NoSideEffect(t *testing.T, caseID string, w *httptest.ResponseRecorder, before, after 境界統合DB副作用スナップショット) {
	t.Helper()
	assertErrorNoSideEffect(t, caseID, w, before, after, http.StatusBadRequest, "invalid request")
}

// assert404NoSideEffect は 404エラー時の共通契約を検証する。
// not found の返却と、副作用DBスナップショット不変を固定する。
func assert404NoSideEffect(t *testing.T, caseID string, w *httptest.ResponseRecorder, before, after 境界統合DB副作用スナップショット) {
	t.Helper()
	assertErrorNoSideEffect(t, caseID, w, before, after, http.StatusNotFound, "not found")
}

// assert409NoSideEffect は 409エラー時の共通契約を検証する。
// amount conflict の返却と、副作用DBスナップショット不変を固定する。
func assert409NoSideEffect(t *testing.T, caseID string, w *httptest.ResponseRecorder, before, after 境界統合DB副作用スナップショット) {
	t.Helper()
	assertErrorNoSideEffect(t, caseID, w, before, after, http.StatusConflict, "amount conflict")
}

// assertErrorNoSideEffect は エラー系共通契約（HTTP分類/主要項目/副作用不変）を検証する。
func assertErrorNoSideEffect(
	t *testing.T,
	caseID string,
	w *httptest.ResponseRecorder,
	before, after 境界統合DB副作用スナップショット,
	expectedStatus int,
	expectedMessage string,
) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Fatalf("[%s] expected %d, got %d body=%s", caseID, expectedStatus, w.Code, strings.TrimSpace(w.Body.String()))
	}
	var body struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("[%s] failed to decode error response: %v", caseID, err)
	}
	if body.Message != expectedMessage {
		t.Fatalf("[%s] expected message=%s, got %s", caseID, expectedMessage, body.Message)
	}
	if before != after {
		t.Fatalf("[%s] expected no db side effects, before=%+v after=%+v", caseID, before, after)
	}
}

// snapshot境界統合副作用 は副作用DB観測の比較用スナップショットを取得する。
// orders/order_items/payments 件数と inventory状態(on_hand/reserved/available) を採取する。
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
	s.inventory = inventoryStateBySKU(t, kit, "sku-1")
	return s
}

// orderStatusByID は orders テーブルの状態列を直接観測する。
// API応答ではなく永続状態の確認に使う。
func orderStatusByID(t *testing.T, kit *境界統合Testkit, orderID string) string {
	t.Helper()

	var status string
	if err := kit.DB.QueryRow(`SELECT status FROM orders WHERE id = $1`, orderID).Scan(&status); err != nil {
		t.Fatalf("failed to fetch order status: %v", err)
	}
	return status
}

// paymentsCountByOrder は order_id に紐づく決済件数を観測する。
// 冪等性と副作用件数の固定に使う。
func paymentsCountByOrder(t *testing.T, kit *境界統合Testkit, orderID string) int {
	t.Helper()

	var count int
	if err := kit.DB.QueryRow(`SELECT COUNT(*) FROM payments WHERE order_id = $1`, orderID).Scan(&count); err != nil {
		t.Fatalf("failed to count payments by order: %v", err)
	}
	return count
}

type inventoryState struct {
	OnHand    int
	Reserved  int
	Available int
}

// inventoryStateBySKU は在庫の3値観測を取得する。
// Available は DB列ではなく OnHand-Reserved で導出して検証軸を揃える。
func inventoryStateBySKU(t *testing.T, kit *境界統合Testkit, sku string) inventoryState {
	t.Helper()

	var s inventoryState
	if err := kit.DB.QueryRow(`SELECT on_hand, reserved FROM inventory WHERE sku = $1`, sku).Scan(&s.OnHand, &s.Reserved); err != nil {
		t.Fatalf("failed to fetch inventory state: %v", err)
	}
	s.Available = s.OnHand - s.Reserved
	return s
}

// assertInventoryReserveTransition は注文作成時の在庫遷移を検証する。
// on_hand 不変、reserved 増、available 減、および恒等式を固定する。
func assertInventoryReserveTransition(t *testing.T, point string, before, after inventoryState, reserveQty int) {
	t.Helper()

	if after.OnHand != before.OnHand {
		t.Fatalf("%s: on_hand must remain unchanged, before=%d after=%d", point, before.OnHand, after.OnHand)
	}
	if after.Reserved != before.Reserved+reserveQty {
		t.Fatalf("%s: reserved must increase by %d, before=%d after=%d", point, reserveQty, before.Reserved, after.Reserved)
	}
	if after.Available != before.Available-reserveQty {
		t.Fatalf("%s: available must decrease by %d, before=%d after=%d", point, reserveQty, before.Available, after.Available)
	}
	if after.Available != after.OnHand-after.Reserved {
		t.Fatalf("%s: available identity broken, on_hand=%d reserved=%d available=%d", point, after.OnHand, after.Reserved, after.Available)
	}
}

// assertInventoryUnchanged は比較時点間で在庫3値が不変であることを検証する。
// 決済確定時の在庫非減算ルールの固定に使う。
func assertInventoryUnchanged(t *testing.T, point string, before, after inventoryState) {
	t.Helper()

	if before != after {
		t.Fatalf("%s: inventory state must remain unchanged, before=%+v after=%+v", point, before, after)
	}
}
