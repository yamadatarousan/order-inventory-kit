package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"order-inventory-kit/internal/domain"
	"order-inventory-kit/internal/usecase"
)

// このテストは OrderHandler のHTTP境界仕様を固定する。
// 仕様対象: 正常系と不正入力/未存在/金額不一致のHTTPステータス分類（200/400/404/409）。
// 根拠: UseCase 実装が変わっても外部境界の振る舞い互換を維持するため。
type stubUsecase struct {
	createOrderFunc    func(input usecase.CreateOrderInput) (usecase.CreateOrderOutput, error)
	getOrderFunc       func(id string) (domain.Order, bool)
	cancelOrderFunc    func(id string) (domain.Order, error)
	confirmPaymentFunc func(input usecase.ConfirmPaymentInput) (usecase.ConfirmPaymentOutput, error)
}

func (s *stubUsecase) CreateOrder(input usecase.CreateOrderInput) (usecase.CreateOrderOutput, error) {
	return s.createOrderFunc(input)
}

func (s *stubUsecase) GetOrder(id string) (domain.Order, bool) {
	return s.getOrderFunc(id)
}

func (s *stubUsecase) CancelOrder(id string) (domain.Order, error) {
	return s.cancelOrderFunc(id)
}

func (s *stubUsecase) ConfirmPayment(input usecase.ConfirmPaymentInput) (usecase.ConfirmPaymentOutput, error) {
	return s.confirmPaymentFunc(input)
}

func setupRouter(t *testing.T, uc *stubUsecase) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewOrderHandler(uc)
	h.RegisterRoutes(r)
	return r
}

func TestCreateOrder_正常系(t *testing.T) {
	uc := &stubUsecase{
		createOrderFunc: func(input usecase.CreateOrderInput) (usecase.CreateOrderOutput, error) {
			return usecase.CreateOrderOutput{OrderID: "order-1", Status: domain.OrderStatusAccepted}, nil
		},
		getOrderFunc:    func(id string) (domain.Order, bool) { return domain.Order{}, false },
		cancelOrderFunc: func(id string) (domain.Order, error) { return domain.Order{}, nil },
		confirmPaymentFunc: func(input usecase.ConfirmPaymentInput) (usecase.ConfirmPaymentOutput, error) {
			return usecase.ConfirmPaymentOutput{}, nil
		},
	}
	r := setupRouter(t, uc)

	body := map[string]any{
		"customerId": "c-1",
		"items":      []map[string]any{{"sku": "sku-1", "quantity": 1, "unitPrice": 100}},
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCreateOrder_不正な入力(t *testing.T) {
	uc := &stubUsecase{
		createOrderFunc: func(input usecase.CreateOrderInput) (usecase.CreateOrderOutput, error) {
			return usecase.CreateOrderOutput{}, errors.New("invalid request")
		},
		getOrderFunc:    func(id string) (domain.Order, bool) { return domain.Order{}, false },
		cancelOrderFunc: func(id string) (domain.Order, error) { return domain.Order{}, nil },
		confirmPaymentFunc: func(input usecase.ConfirmPaymentInput) (usecase.ConfirmPaymentOutput, error) {
			return usecase.ConfirmPaymentOutput{}, nil
		},
	}
	r := setupRouter(t, uc)

	payload, _ := json.Marshal(map[string]any{"customerId": "c-1", "items": []any{}})
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateOrder_価格不一致(t *testing.T) {
	uc := &stubUsecase{
		createOrderFunc: func(input usecase.CreateOrderInput) (usecase.CreateOrderOutput, error) {
			return usecase.CreateOrderOutput{}, usecase.ErrCreateOrderPriceConflict
		},
		getOrderFunc:    func(id string) (domain.Order, bool) { return domain.Order{}, false },
		cancelOrderFunc: func(id string) (domain.Order, error) { return domain.Order{}, nil },
		confirmPaymentFunc: func(input usecase.ConfirmPaymentInput) (usecase.ConfirmPaymentOutput, error) {
			return usecase.ConfirmPaymentOutput{}, nil
		},
	}
	r := setupRouter(t, uc)

	payload, _ := json.Marshal(map[string]any{
		"customerId": "c-1",
		"items":      []map[string]any{{"sku": "sku-1", "quantity": 1, "unitPrice": 101}},
	})
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestCreateOrder_顧客不整合(t *testing.T) {
	uc := &stubUsecase{
		createOrderFunc: func(input usecase.CreateOrderInput) (usecase.CreateOrderOutput, error) {
			return usecase.CreateOrderOutput{}, usecase.ErrCreateOrderInvalidCustomer
		},
		getOrderFunc:    func(id string) (domain.Order, bool) { return domain.Order{}, false },
		cancelOrderFunc: func(id string) (domain.Order, error) { return domain.Order{}, nil },
		confirmPaymentFunc: func(input usecase.ConfirmPaymentInput) (usecase.ConfirmPaymentOutput, error) {
			return usecase.ConfirmPaymentOutput{}, nil
		},
	}
	r := setupRouter(t, uc)

	payload, _ := json.Marshal(map[string]any{
		"customerId": "missing-customer",
		"items":      []map[string]any{{"sku": "sku-1", "quantity": 1, "unitPrice": 100}},
	})
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetOrder_存在する場合(t *testing.T) {
	uc := &stubUsecase{
		createOrderFunc: func(input usecase.CreateOrderInput) (usecase.CreateOrderOutput, error) {
			return usecase.CreateOrderOutput{}, nil
		},
		getOrderFunc: func(id string) (domain.Order, bool) {
			return domain.Order{ID: id, Status: domain.OrderStatusAccepted}, true
		},
		cancelOrderFunc: func(id string) (domain.Order, error) { return domain.Order{}, nil },
		confirmPaymentFunc: func(input usecase.ConfirmPaymentInput) (usecase.ConfirmPaymentOutput, error) {
			return usecase.ConfirmPaymentOutput{}, nil
		},
	}
	r := setupRouter(t, uc)

	req := httptest.NewRequest(http.MethodGet, "/orders/order-1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetOrder_存在しない場合(t *testing.T) {
	uc := &stubUsecase{
		createOrderFunc: func(input usecase.CreateOrderInput) (usecase.CreateOrderOutput, error) {
			return usecase.CreateOrderOutput{}, nil
		},
		getOrderFunc:    func(id string) (domain.Order, bool) { return domain.Order{}, false },
		cancelOrderFunc: func(id string) (domain.Order, error) { return domain.Order{}, nil },
		confirmPaymentFunc: func(input usecase.ConfirmPaymentInput) (usecase.ConfirmPaymentOutput, error) {
			return usecase.ConfirmPaymentOutput{}, nil
		},
	}
	r := setupRouter(t, uc)

	req := httptest.NewRequest(http.MethodGet, "/orders/missing", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCancelOrder_存在しない場合(t *testing.T) {
	uc := &stubUsecase{
		createOrderFunc: func(input usecase.CreateOrderInput) (usecase.CreateOrderOutput, error) {
			return usecase.CreateOrderOutput{}, nil
		},
		getOrderFunc:    func(id string) (domain.Order, bool) { return domain.Order{}, false },
		cancelOrderFunc: func(id string) (domain.Order, error) { return domain.Order{}, errors.New("not found") },
		confirmPaymentFunc: func(input usecase.ConfirmPaymentInput) (usecase.ConfirmPaymentOutput, error) {
			return usecase.ConfirmPaymentOutput{}, nil
		},
	}
	r := setupRouter(t, uc)

	req := httptest.NewRequest(http.MethodPost, "/orders/missing/cancel", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestConfirmPayment_注文が存在しない(t *testing.T) {
	uc := &stubUsecase{
		createOrderFunc: func(input usecase.CreateOrderInput) (usecase.CreateOrderOutput, error) {
			return usecase.CreateOrderOutput{}, nil
		},
		getOrderFunc:    func(id string) (domain.Order, bool) { return domain.Order{}, false },
		cancelOrderFunc: func(id string) (domain.Order, error) { return domain.Order{}, nil },
		confirmPaymentFunc: func(input usecase.ConfirmPaymentInput) (usecase.ConfirmPaymentOutput, error) {
			return usecase.ConfirmPaymentOutput{}, usecase.ErrConfirmPaymentNotFound
		},
	}
	r := setupRouter(t, uc)

	payload, _ := json.Marshal(map[string]any{"orderId": "missing", "amount": 100, "idempotencyKey": "k-1"})
	req := httptest.NewRequest(http.MethodPost, "/payments/confirm", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestConfirmPayment_不正な入力(t *testing.T) {
	uc := &stubUsecase{
		createOrderFunc: func(input usecase.CreateOrderInput) (usecase.CreateOrderOutput, error) {
			return usecase.CreateOrderOutput{}, nil
		},
		getOrderFunc:    func(id string) (domain.Order, bool) { return domain.Order{}, false },
		cancelOrderFunc: func(id string) (domain.Order, error) { return domain.Order{}, nil },
		confirmPaymentFunc: func(input usecase.ConfirmPaymentInput) (usecase.ConfirmPaymentOutput, error) {
			return usecase.ConfirmPaymentOutput{}, nil
		},
	}
	r := setupRouter(t, uc)

	payload, _ := json.Marshal(map[string]any{"orderId": "", "amount": 0, "idempotencyKey": ""})
	req := httptest.NewRequest(http.MethodPost, "/payments/confirm", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestConfirmPayment_金額不一致(t *testing.T) {
	uc := &stubUsecase{
		createOrderFunc: func(input usecase.CreateOrderInput) (usecase.CreateOrderOutput, error) {
			return usecase.CreateOrderOutput{}, nil
		},
		getOrderFunc:    func(id string) (domain.Order, bool) { return domain.Order{}, false },
		cancelOrderFunc: func(id string) (domain.Order, error) { return domain.Order{}, nil },
		confirmPaymentFunc: func(input usecase.ConfirmPaymentInput) (usecase.ConfirmPaymentOutput, error) {
			return usecase.ConfirmPaymentOutput{}, usecase.ErrConfirmPaymentAmountConflict
		},
	}
	r := setupRouter(t, uc)

	payload, _ := json.Marshal(map[string]any{"orderId": "order-1", "amount": 101, "idempotencyKey": "k-1"})
	req := httptest.NewRequest(http.MethodPost, "/payments/confirm", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestConfirmPayment_ユースケースが不正入力を返す(t *testing.T) {
	uc := &stubUsecase{
		createOrderFunc: func(input usecase.CreateOrderInput) (usecase.CreateOrderOutput, error) {
			return usecase.CreateOrderOutput{}, nil
		},
		getOrderFunc:    func(id string) (domain.Order, bool) { return domain.Order{}, false },
		cancelOrderFunc: func(id string) (domain.Order, error) { return domain.Order{}, nil },
		confirmPaymentFunc: func(input usecase.ConfirmPaymentInput) (usecase.ConfirmPaymentOutput, error) {
			return usecase.ConfirmPaymentOutput{}, usecase.ErrConfirmPaymentInvalidRequest
		},
	}
	r := setupRouter(t, uc)

	payload, _ := json.Marshal(map[string]any{"orderId": "order-1", "amount": 100, "idempotencyKey": "k-1"})
	req := httptest.NewRequest(http.MethodPost, "/payments/confirm", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
