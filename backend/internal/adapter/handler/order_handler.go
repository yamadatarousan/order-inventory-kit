// Package handler はHTTPハンドラを定義する。
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"order-inventory-kit/internal/domain"
	"order-inventory-kit/internal/usecase"
)

type orderUsecase interface {
	CreateOrder(input usecase.CreateOrderInput) (usecase.CreateOrderOutput, error)
	GetOrder(id string) (domain.Order, bool)
	CancelOrder(id string) (domain.Order, error)
	ConfirmPayment(input usecase.ConfirmPaymentInput) (usecase.ConfirmPaymentOutput, error)
}

// OrderHandler は注文APIのハンドラ。
type OrderHandler struct {
	uc orderUsecase
}

// NewOrderHandler は注文ハンドラを作成する。
func NewOrderHandler(uc orderUsecase) *OrderHandler {
	return &OrderHandler{uc: uc}
}

// createOrderRequest は注文作成の入力。
type createOrderRequest struct {
	CustomerID string                   `json:"customerId"`
	Items      []createOrderRequestItem `json:"items"`
}

type createOrderRequestItem struct {
	SKU       string `json:"sku"`
	Quantity  int    `json:"quantity"`
	UnitPrice *int   `json:"unitPrice"`
}

// createOrderResponse は注文作成の出力。
type createOrderResponse struct {
	OrderID string             `json:"orderId"`
	Status  domain.OrderStatus `json:"status"`
}

// confirmPaymentRequest は決済確定の入力。
type confirmPaymentRequest struct {
	OrderID        string `json:"orderId"`
	Amount         int    `json:"amount"`
	IdempotencyKey string `json:"idempotencyKey"`
}

// confirmPaymentResponse は決済確定の出力。
type confirmPaymentResponse struct {
	OrderID       string `json:"orderId"`
	PaymentStatus string `json:"paymentStatus"`
}

// RegisterRoutes はルーティングを登録する。
func (h *OrderHandler) RegisterRoutes(r *gin.Engine) {
	r.POST("/orders", h.createOrder)
	r.GET("/orders/:orderId", h.getOrder)
	r.POST("/orders/:orderId/cancel", h.cancelOrder)
	r.POST("/payments/confirm", h.confirmPayment)
}

// createOrder は注文を作成する。
func (h *OrderHandler) createOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}

	items := make([]domain.OrderItem, 0, len(req.Items))
	quotedUnitPrices := make(map[string]int, len(req.Items))
	for _, item := range req.Items {
		if item.UnitPrice == nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
			return
		}
		items = append(items, domain.OrderItem{
			SKU:      item.SKU,
			Quantity: item.Quantity,
		})
		quotedUnitPrices[item.SKU] = *item.UnitPrice
	}

	out, err := h.uc.CreateOrder(usecase.CreateOrderInput{
		CustomerID:       req.CustomerID,
		Items:            items,
		QuotedUnitPrices: quotedUnitPrices,
	})
	if err != nil {
		status, message := classifyCreateOrderError(err)
		c.JSON(status, gin.H{"message": message})
		return
	}
	c.JSON(http.StatusOK, createOrderResponse{OrderID: out.OrderID, Status: out.Status})
}

// getOrder は注文を取得する。
func (h *OrderHandler) getOrder(c *gin.Context) {
	id := c.Param("orderId")
	order, ok := h.uc.GetOrder(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"message": "not found"})
		return
	}
	c.JSON(http.StatusOK, order)
}

// cancelOrder は注文をキャンセルする。
func (h *OrderHandler) cancelOrder(c *gin.Context) {
	id := c.Param("orderId")
	order, err := h.uc.CancelOrder(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "not found"})
		return
	}
	c.JSON(http.StatusOK, order)
}

// confirmPayment は決済を確定する。
func (h *OrderHandler) confirmPayment(c *gin.Context) {
	var req confirmPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	if req.OrderID == "" || req.IdempotencyKey == "" || req.Amount < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	out, err := h.uc.ConfirmPayment(usecase.ConfirmPaymentInput{OrderID: req.OrderID, Amount: req.Amount, IdempotencyKey: req.IdempotencyKey})
	if err != nil {
		status, message := classifyConfirmPaymentError(err)
		c.JSON(status, gin.H{"message": message})
		return
	}
	c.JSON(http.StatusOK, confirmPaymentResponse{OrderID: out.OrderID, PaymentStatus: out.PaymentStatus})
}

func classifyConfirmPaymentError(err error) (int, string) {
	switch {
	case errors.Is(err, usecase.ErrConfirmPaymentNotFound):
		return http.StatusNotFound, "not found"
	case errors.Is(err, usecase.ErrConfirmPaymentAmountConflict):
		return http.StatusConflict, "amount conflict"
	default:
		return http.StatusBadRequest, "invalid request"
	}
}

func classifyCreateOrderError(err error) (int, string) {
	switch {
	case errors.Is(err, usecase.ErrCreateOrderPriceConflict):
		return http.StatusConflict, "price conflict"
	default:
		return http.StatusBadRequest, "invalid request"
	}
}
