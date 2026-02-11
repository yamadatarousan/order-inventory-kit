// Package domain はコアのエンティティとルールを定義する。
package domain

import "errors"

// OrderStatus は注文の状態を表す。
type OrderStatus string

// 注文状態の一覧。
const (
	OrderStatusAccepted  OrderStatus = "accepted"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusCanceled  OrderStatus = "canceled"
)

// OrderItem は注文内の商品を表す。
type OrderItem struct {
	SKU      string
	Quantity int
}

// Order は注文の中核エンティティ。
type Order struct {
	ID         string
	CustomerID string
	Status     OrderStatus
	Items      []OrderItem
}

// NewOrder は入力を検証して Order を作成する。
func NewOrder(id, customerID string, items []OrderItem) (Order, error) {
	if id == "" {
		return Order{}, errors.New("id is required")
	}
	if customerID == "" {
		return Order{}, errors.New("customer id is required")
	}
	if len(items) == 0 {
		return Order{}, errors.New("items is required")
	}
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item.SKU == "" {
			return Order{}, errors.New("sku is required")
		}
		if item.Quantity < 1 {
			return Order{}, errors.New("quantity must be >= 1")
		}
		if _, exists := seen[item.SKU]; exists {
			return Order{}, errors.New("duplicate sku")
		}
		seen[item.SKU] = struct{}{}
	}
	return Order{ID: id, CustomerID: customerID, Status: OrderStatusAccepted, Items: items}, nil
}
