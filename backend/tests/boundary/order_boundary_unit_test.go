package boundary_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"order-inventory-kit/internal/adapter/handler"
	"order-inventory-kit/internal/domain"
	"order-inventory-kit/internal/usecase"
)

// このテストは 境界単体テスト（Stub前提）としてHTTP境界の最小仕様を固定する。
// 仕様対象: POST/GET連続呼び出し時の状態整合と、404/400の分類。
// 根拠: 統合境界テストとは分離し、HTTP変換/分類の局所回帰を早く検出するため。
type 境界前提用UsecaseStub struct {
	orders map[string]domain.Order
}

func new境界前提用UsecaseStub() *境界前提用UsecaseStub {
	return &境界前提用UsecaseStub{
		orders: make(map[string]domain.Order),
	}
}

func (s *境界前提用UsecaseStub) CreateOrder(input usecase.CreateOrderInput) (usecase.CreateOrderOutput, error) {
	id := "order-1"
	s.orders[id] = domain.Order{
		ID:     id,
		Status: domain.OrderStatusConfirmed,
		Items:  input.Items,
	}
	return usecase.CreateOrderOutput{
		OrderID: id,
		Status:  domain.OrderStatusAccepted,
	}, nil
}

func (s *境界前提用UsecaseStub) GetOrder(id string) (domain.Order, bool) {
	order, ok := s.orders[id]
	return order, ok
}

func (s *境界前提用UsecaseStub) CancelOrder(id string) (domain.Order, error) {
	order, ok := s.orders[id]
	if !ok {
		return domain.Order{}, errors.New("not found")
	}
	order.Status = domain.OrderStatusCanceled
	s.orders[id] = order
	return order, nil
}

func (s *境界前提用UsecaseStub) ConfirmPayment(input usecase.ConfirmPaymentInput) (usecase.ConfirmPaymentOutput, error) {
	if input.OrderID == "" || input.IdempotencyKey == "" || input.Amount < 1 {
		return usecase.ConfirmPaymentOutput{}, errors.New("invalid request")
	}
	if _, ok := s.orders[input.OrderID]; !ok {
		return usecase.ConfirmPaymentOutput{}, errors.New("not found")
	}
	return usecase.ConfirmPaymentOutput{
		OrderID:       input.OrderID,
		PaymentStatus: "confirmed",
	}, nil
}

func setupBoundaryRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewOrderHandler(new境界前提用UsecaseStub())
	h.RegisterRoutes(r)
	return r
}

func TestBoundary_POST_ordersがacceptedならGET_orders_idでconfirmedを観測できる(t *testing.T) {
	r := setupBoundaryRouter(t)

	payload, _ := json.Marshal(map[string]any{
		"customerId": "c-1",
		"items":      []map[string]any{{"sku": "sku-1", "quantity": 1}},
	})
	postReq := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(payload))
	postReq.Header.Set("Content-Type", "application/json")
	postW := httptest.NewRecorder()
	r.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", postW.Code)
	}

	var postRes struct {
		OrderID string `json:"orderId"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(postW.Body.Bytes(), &postRes); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if postRes.Status != "accepted" {
		t.Fatalf("expected accepted, got %s", postRes.Status)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/orders/"+postRes.OrderID, nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getW.Code)
	}

	var getRes struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(getW.Body.Bytes(), &getRes); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if getRes.Status != "confirmed" {
		t.Fatalf("expected confirmed, got %s", getRes.Status)
	}
}

func TestBoundary_GET_orders_存在しないIDは404(t *testing.T) {
	r := setupBoundaryRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/orders/missing", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestBoundary_POST_payments_confirm_不正入力は400(t *testing.T) {
	r := setupBoundaryRouter(t)

	payload, _ := json.Marshal(map[string]any{
		"orderId":        "",
		"amount":         0,
		"idempotencyKey": "",
	})
	req := httptest.NewRequest(http.MethodPost, "/payments/confirm", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
