// Package db はDB実装を提供する。
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

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
		_, err = tx.Exec(`
			INSERT INTO inventory_reservations (order_id, sku, quantity)
			VALUES ($1, $2, $3)
		`, order.ID, item.SKU, item.Quantity)
		if err != nil {
			return err
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

// GetTotalAmount は注文合計（quantity * unit_price の合計）を取得する。
func (r *OrderRepository) GetTotalAmount(id string) (int, bool) {
	var exists bool
	if err := r.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM orders WHERE id = $1)
	`, id).Scan(&exists); err != nil {
		return 0, false
	}
	if !exists {
		return 0, false
	}

	var total int
	if err := r.db.QueryRow(`
		SELECT COALESCE(SUM(quantity * COALESCE(unit_price, 0)), 0)
		FROM order_items
		WHERE order_id = $1
	`, id).Scan(&total); err != nil {
		return 0, false
	}
	return total, true
}

// Update は注文を更新する。
func (r *OrderRepository) Update(order domain.Order) error {
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
		UPDATE orders SET status = $1 WHERE id = $2
	`, order.Status, order.ID)
	if err != nil {
		return err
	}
	if order.Status == domain.OrderStatusCanceled {
		_, err = tx.Exec(`
			DELETE FROM inventory_reservations WHERE order_id = $1
		`, order.ID)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

// ExpireAcceptedOrders は期限切れの accepted 注文を canceled に更新し、引当を戻す。
// 期限判定は created_at < cutoff を使う。
func (r *OrderRepository) ExpireAcceptedOrders(cutoff time.Time) (int, error) {
	tx, err := r.db.BeginTx(context.Background(), nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	rows, err := tx.Query(`
		SELECT id
		FROM orders
		WHERE status = $1 AND created_at < $2
		ORDER BY id
	`, domain.OrderStatusAccepted, cutoff)
	if err != nil {
		return 0, fmt.Errorf("query expired orders: %w", err)
	}
	defer rows.Close()

	var orderIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("scan expired order id: %w", err)
		}
		orderIDs = append(orderIDs, id)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate expired order ids: %w", err)
	}
	if err := rows.Close(); err != nil {
		return 0, fmt.Errorf("close expired order rows: %w", err)
	}

	expiredCount := 0
	for _, orderID := range orderIDs {
		type reservation struct {
			sku      string
			quantity int
		}

		reservationRows, err := tx.Query(`
			SELECT sku, quantity
			FROM inventory_reservations
			WHERE order_id = $1
			ORDER BY sku
		`, orderID)
		if err != nil {
			return expiredCount, fmt.Errorf("query reservations order=%s: %w", orderID, err)
		}

		var reservations []reservation
		for reservationRows.Next() {
			var r reservation
			if err := reservationRows.Scan(&r.sku, &r.quantity); err != nil {
				_ = reservationRows.Close()
				return expiredCount, fmt.Errorf("scan reservation row order=%s: %w", orderID, err)
			}
			reservations = append(reservations, r)
		}
		if err := reservationRows.Err(); err != nil {
			_ = reservationRows.Close()
			return expiredCount, fmt.Errorf("iterate reservations order=%s: %w", orderID, err)
		}
		if err := reservationRows.Close(); err != nil {
			return expiredCount, fmt.Errorf("close reservations order=%s: %w", orderID, err)
		}

		if len(reservations) == 0 {
			return expiredCount, fmt.Errorf("reservation not found for expired order: %s", orderID)
		}
		for _, r := range reservations {
			result, err := tx.Exec(`
				UPDATE inventory
				SET
				  reserved = reserved - $2,
				  quantity = on_hand - (reserved - $2)
				WHERE sku = $1 AND reserved >= $2
			`, r.sku, r.quantity)
			if err != nil {
				return expiredCount, fmt.Errorf("update inventory release order=%s sku=%s: %w", orderID, r.sku, err)
			}
			affected, err := result.RowsAffected()
			if err != nil {
				return expiredCount, fmt.Errorf("rows affected inventory release order=%s sku=%s: %w", orderID, r.sku, err)
			}
			if affected != 1 {
				return expiredCount, fmt.Errorf("inventory release failed for sku=%s quantity=%d", r.sku, r.quantity)
			}
		}

		if _, err := tx.Exec(`DELETE FROM inventory_reservations WHERE order_id = $1`, orderID); err != nil {
			return expiredCount, fmt.Errorf("delete reservations order=%s: %w", orderID, err)
		}
		if _, err := tx.Exec(`UPDATE orders SET status = $1 WHERE id = $2`, domain.OrderStatusCanceled, orderID); err != nil {
			return expiredCount, fmt.Errorf("update order canceled order=%s: %w", orderID, err)
		}
		expiredCount++
	}

	if err := tx.Commit(); err != nil {
		return expiredCount, fmt.Errorf("commit expiration tx: %w", err)
	}
	committed = true
	return expiredCount, nil
}
