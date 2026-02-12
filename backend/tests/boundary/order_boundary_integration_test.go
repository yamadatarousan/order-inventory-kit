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
