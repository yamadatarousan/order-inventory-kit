package domain

import "testing"

func TestNewOrder_正常系(t *testing.T) {
	items := []OrderItem{{SKU: "sku-1", Quantity: 1}}
	order, err := NewOrder("order-1", items)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if order.ID != "order-1" {
		t.Fatalf("expected id order-1, got %s", order.ID)
	}
	if order.Status != OrderStatusAccepted {
		t.Fatalf("expected status accepted, got %s", order.Status)
	}
}

func TestNewOrder_異常系(t *testing.T) {
	cases := []struct {
		name  string
		id    string
		items []OrderItem
	}{
		{name: "idが空", id: "", items: []OrderItem{{SKU: "sku-1", Quantity: 1}}},
		{name: "itemsが空", id: "order-1", items: []OrderItem{}},
		{name: "skuが空", id: "order-1", items: []OrderItem{{SKU: "", Quantity: 1}}},
		{name: "quantityが0", id: "order-1", items: []OrderItem{{SKU: "sku-1", Quantity: 0}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewOrder(tc.id, tc.items)
			if err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestNewOrder_追加仕様_同一SKUの重複は禁止(t *testing.T) {
	items := []OrderItem{{SKU: "sku-1", Quantity: 1}, {SKU: "sku-1", Quantity: 2}}
	_, err := NewOrder("order-1", items)
	if err == nil {
		t.Fatalf("expected error")
	}
}
