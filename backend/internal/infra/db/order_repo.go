// Package db はDB実装を提供する。
package db

import (
	"context"
	"database/sql"
	"errors"

	"order-inventory-kit/internal/domain"
)

// OrderRepository はDBの注文リポジトリ。
type OrderRepository struct {
	db *sql.DB
}

// NewOrderRepository はDBの注文リポジトリを作成する。
func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create は注文を保存する。
func (r *OrderRepository) Create(order domain.Order) error {
	tx, err := r.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.Exec(`
		INSERT INTO orders (id, customer_id, status) VALUES ($1, $2, $3)
	`, order.ID, order.CustomerID, order.Status)
	if err != nil {
		return err
	}
	for _, item := range order.Items {
		result, err := tx.Exec(`
			INSERT INTO order_items (order_id, sku, quantity, unit_price)
			SELECT $1, $2, $3, product_prices.unit_price
			FROM product_prices
			WHERE product_prices.sku = $2
		`, order.ID, item.SKU, item.Quantity)
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows != 1 {
			return errors.New("price not found")
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

// Get は注文を取得する。
func (r *OrderRepository) Get(id string) (domain.Order, bool) {
	row := r.db.QueryRow(`
		SELECT id, customer_id, status FROM orders WHERE id = $1
	`, id)

	var order domain.Order
	if err := row.Scan(&order.ID, &order.CustomerID, &order.Status); err != nil {
		return domain.Order{}, false
	}

	rows, err := r.db.Query(`
		SELECT sku, quantity FROM order_items WHERE order_id = $1 ORDER BY sku
	`, id)
	if err != nil {
		return domain.Order{}, false
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(&item.SKU, &item.Quantity); err != nil {
			return domain.Order{}, false
		}
		order.Items = append(order.Items, item)
	}
	return order, true
}

// Update は注文を更新する。
func (r *OrderRepository) Update(order domain.Order) error {
	_, err := r.db.Exec(`
		UPDATE orders SET status = $1 WHERE id = $2
	`, order.Status, order.ID)
	return err
}
