package boundary_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// このテストは 境界一貫性統合テスト の受け皿を固定する。
// 仕様対象: 実Router+実UseCase+実DB を通した注文作成->決済確定->注文参照の通し挙動。
// 根拠: Stub前提の単体境界テストとは別に、実依存を通した境界一貫性を固定するため。
func TestIntegration_OrderBoundary_注文作成から決済確定と注文参照まで通し検証(t *testing.T) {
	kit := new境界統合Testkit(t)

	createPayload, _ := json.Marshal(map[string]any{
		"customerId": "c-1",
		"items":      []map[string]any{{"sku": "sku-1", "quantity": 1}},
	})
	createReq := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(createPayload))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	kit.Router.ServeHTTP(createW, createReq)

	if createW.Code != http.StatusOK {
		t.Fatalf("expected 200 on create, got %d", createW.Code)
	}

	var createRes struct {
		OrderID string `json:"orderId"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(createW.Body.Bytes(), &createRes); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	if createRes.OrderID != "integration-order-1" {
		t.Fatalf("order id must be deterministic in integration test, got %s", createRes.OrderID)
	}
	if createRes.Status != "accepted" {
		t.Fatalf("expected accepted, got %s", createRes.Status)
	}

	confirmPayload, _ := json.Marshal(map[string]any{
		"orderId":        createRes.OrderID,
		"amount":         100,
		"idempotencyKey": "k-1",
	})
	confirmReq := httptest.NewRequest(http.MethodPost, "/payments/confirm", bytes.NewReader(confirmPayload))
	confirmReq.Header.Set("Content-Type", "application/json")
	confirmW := httptest.NewRecorder()
	kit.Router.ServeHTTP(confirmW, confirmReq)
	if confirmW.Code != http.StatusOK {
		t.Fatalf("expected 200 on confirm, got %d", confirmW.Code)
	}

	var confirmRes struct {
		OrderID       string `json:"orderId"`
		PaymentStatus string `json:"paymentStatus"`
	}
	if err := json.Unmarshal(confirmW.Body.Bytes(), &confirmRes); err != nil {
		t.Fatalf("failed to decode confirm response: %v", err)
	}
	if confirmRes.OrderID != createRes.OrderID {
		t.Fatalf("confirm order id mismatch: create=%s confirm=%s", createRes.OrderID, confirmRes.OrderID)
	}
	if confirmRes.PaymentStatus != "confirmed" {
		t.Fatalf("expected paymentStatus=confirmed, got %s", confirmRes.PaymentStatus)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/orders/"+createRes.OrderID, nil)
	getW := httptest.NewRecorder()
	kit.Router.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200 on get, got %d", getW.Code)
	}

	var getRes struct {
		ID         string `json:"id"`
		CustomerID string `json:"customerId"`
		Status     string `json:"status"`
		Items      []struct {
			SKU      string `json:"sku"`
			Quantity int    `json:"quantity"`
		} `json:"items"`
	}
	if err := json.Unmarshal(getW.Body.Bytes(), &getRes); err != nil {
		t.Fatalf("failed to decode get response: %v", err)
	}
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
