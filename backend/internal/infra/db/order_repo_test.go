package db

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"order-inventory-kit/internal/domain"
)

func testQuotedUnitPrices(items []domain.OrderItem) map[string]int {
	prices := make(map[string]int, len(items))
	for _, item := range items {
		switch item.SKU {
		case "sku-1":
			prices[item.SKU] = 100
		case "sku-2":
			prices[item.SKU] = 80
		case "sku-3":
			prices[item.SKU] = 50
		default:
			prices[item.SKU] = 100
		}
	}
	return prices
}

// このテストは OrderRepository の永続化仕様を固定する。
// 仕様対象: 注文の作成・取得・更新で Order の状態と明細が正しく保存されること。
// 根拠: 保存方式を変更しても注文状態遷移の基盤となるデータ整合を維持するため。
func TestOrderRepository_作成と取得(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	repo := NewOrderRepository(db)
	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 2}})

	if err := repo.Create(order, testQuotedUnitPrices(order.Items)); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	got, ok := repo.Get("order-1")
	if !ok {
		t.Fatalf("expected order to exist")
	}
	if got.Status != domain.OrderStatusAccepted {
		t.Fatalf("expected accepted, got %s", got.Status)
	}
	if got.CustomerID != "c-1" {
		t.Fatalf("expected customer id c-1, got %s", got.CustomerID)
	}
	if len(got.Items) != 1 || got.Items[0].SKU != "sku-1" {
		t.Fatalf("expected items to be saved")
	}
}

func TestOrderRepository_更新(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	repo := NewOrderRepository(db)
	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 2}})
	_ = repo.Create(order, testQuotedUnitPrices(order.Items))

	order.Status = domain.OrderStatusCanceled
	if err := repo.Update(order); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	got, _ := repo.Get("order-1")
	if got.Status != domain.OrderStatusCanceled {
		t.Fatalf("expected canceled, got %s", got.Status)
	}
}

func TestOrderRepository_作成時に単価スナップショットを保存する(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	repo := NewOrderRepository(db)
	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 2}})

	if err := repo.Create(order, testQuotedUnitPrices(order.Items)); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var savedUnitPrice sql.NullInt64
	if err := db.QueryRow(`SELECT unit_price FROM order_items WHERE order_id = $1 AND sku = $2`, "order-1", "sku-1").Scan(&savedUnitPrice); err != nil {
		t.Fatalf("failed to load unit_price: %v", err)
	}
	if !savedUnitPrice.Valid {
		t.Fatalf("expected unit_price to be set")
	}
	if savedUnitPrice.Int64 != 100 {
		t.Fatalf("expected unit_price 100, got %d", savedUnitPrice.Int64)
	}

	if _, err := db.Exec(`UPDATE products SET unit_price = 999 WHERE sku = $1`, "sku-1"); err != nil {
		t.Fatalf("failed to update product master price: %v", err)
	}
	defer func() {
		_, _ = db.Exec(`UPDATE products SET unit_price = 100 WHERE sku = $1`, "sku-1")
	}()

	var snapshotPrice int
	if err := db.QueryRow(`SELECT unit_price FROM order_items WHERE order_id = $1 AND sku = $2`, "order-1", "sku-1").Scan(&snapshotPrice); err != nil {
		t.Fatalf("failed to reload snapshot unit_price: %v", err)
	}
	if snapshotPrice != 100 {
		t.Fatalf("snapshot unit_price must remain 100, got %d", snapshotPrice)
	}
}

func TestOrderRepository_作成時に在庫引当台帳を保存する(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	repo := NewOrderRepository(db)
	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{
		{SKU: "sku-1", Quantity: 2},
		{SKU: "sku-2", Quantity: 1},
	})

	if err := repo.Create(order, testQuotedUnitPrices(order.Items)); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	rows, err := db.Query(`
		SELECT sku, quantity
		FROM inventory_reservations
		WHERE order_id = $1
		ORDER BY sku
	`, "order-1")
	if err != nil {
		t.Fatalf("failed to query inventory reservations: %v", err)
	}
	defer rows.Close()

	type reservation struct {
		sku      string
		quantity int
	}
	var got []reservation
	for rows.Next() {
		var r reservation
		if err := rows.Scan(&r.sku, &r.quantity); err != nil {
			t.Fatalf("failed to scan reservation row: %v", err)
		}
		got = append(got, r)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 reservation rows, got %d", len(got))
	}
	if got[0] != (reservation{sku: "sku-1", quantity: 2}) {
		t.Fatalf("unexpected reservation row[0]: %+v", got[0])
	}
	if got[1] != (reservation{sku: "sku-2", quantity: 1}) {
		t.Fatalf("unexpected reservation row[1]: %+v", got[1])
	}
}

func TestOrderRepository_キャンセル更新時に在庫引当台帳を解放する(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	repo := NewOrderRepository(db)
	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 2}})
	if err := repo.Create(order, testQuotedUnitPrices(order.Items)); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var before int
	if err := db.QueryRow(`SELECT COUNT(*) FROM inventory_reservations WHERE order_id = $1`, "order-1").Scan(&before); err != nil {
		t.Fatalf("failed to count reservations before cancel: %v", err)
	}
	if before != 1 {
		t.Fatalf("expected one reservation row before cancel, got %d", before)
	}

	order.Status = domain.OrderStatusCanceled
	if err := repo.Update(order); err != nil {
		t.Fatalf("expected no error on cancel update, got %v", err)
	}

	var after int
	if err := db.QueryRow(`SELECT COUNT(*) FROM inventory_reservations WHERE order_id = $1`, "order-1").Scan(&after); err != nil {
		t.Fatalf("failed to count reservations after cancel: %v", err)
	}
	if after != 0 {
		t.Fatalf("expected reservation rows to be released on cancel, got %d", after)
	}
}

func TestOrderRepository_価格情報が存在しないSKUを含む注文は失敗する(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	repo := NewOrderRepository(db)
	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "missing-sku", Quantity: 1}})

	if err := repo.Create(order, testQuotedUnitPrices(order.Items)); err == nil {
		t.Fatalf("expected create to fail when price is missing")
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM orders WHERE id = $1`, "order-1").Scan(&count); err != nil {
		t.Fatalf("failed to count order rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("order row must not be persisted on failure, got %d", count)
	}

	var reservationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM inventory_reservations WHERE order_id = $1`, "order-1").Scan(&reservationCount); err != nil {
		t.Fatalf("failed to count reservation rows: %v", err)
	}
	if reservationCount != 0 {
		t.Fatalf("reservation rows must not be persisted on failure, got %d", reservationCount)
	}
}

func TestOrderRepository_提示価格が不一致の注文は失敗する(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	repo := NewOrderRepository(db)
	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 1}})

	err := repo.Create(order, map[string]int{"sku-1": 101})
	if err == nil {
		t.Fatalf("expected create to fail when quoted price mismatches")
	}
	if !errors.Is(err, domain.ErrPriceConflict) {
		t.Fatalf("expected price conflict, got %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM orders WHERE id = $1`, "order-1").Scan(&count); err != nil {
		t.Fatalf("failed to count order rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("order row must not be persisted on failure, got %d", count)
	}

	var reservationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM inventory_reservations WHERE order_id = $1`, "order-1").Scan(&reservationCount); err != nil {
		t.Fatalf("failed to count reservation rows: %v", err)
	}
	if reservationCount != 0 {
		t.Fatalf("reservation rows must not be persisted on failure, got %d", reservationCount)
	}
}

func TestOrderRepository_販売停止SKUを含む注文は失敗する(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	if _, err := db.Exec(`UPDATE products SET is_active = FALSE WHERE sku = $1`, "sku-1"); err != nil {
		t.Fatalf("failed to deactivate product: %v", err)
	}
	defer func() {
		_, _ = db.Exec(`UPDATE products SET is_active = TRUE WHERE sku = $1`, "sku-1")
	}()

	repo := NewOrderRepository(db)
	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 1}})

	if err := repo.Create(order, testQuotedUnitPrices(order.Items)); err == nil {
		t.Fatalf("expected create to fail when product is inactive")
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM orders WHERE id = $1`, "order-1").Scan(&count); err != nil {
		t.Fatalf("failed to count order rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("order row must not be persisted on failure, got %d", count)
	}

	var reservationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM inventory_reservations WHERE order_id = $1`, "order-1").Scan(&reservationCount); err != nil {
		t.Fatalf("failed to count reservation rows: %v", err)
	}
	if reservationCount != 0 {
		t.Fatalf("reservation rows must not be persisted on failure, got %d", reservationCount)
	}
}

func TestOrderRepository_期限切れ_acceptedは引当を戻してcanceledに更新する(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	repo := NewOrderRepository(db)
	if _, err := db.Exec(`
		INSERT INTO inventory (sku, quantity, on_hand, reserved)
		VALUES ('sku-1', 100, 100, 0)
	`); err != nil {
		t.Fatalf("failed to seed inventory row: %v", err)
	}
	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 2}})
	if err := repo.Create(order, testQuotedUnitPrices(order.Items)); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, err := db.Exec(`
		UPDATE inventory
		SET reserved = 2, quantity = on_hand - 2
		WHERE sku = 'sku-1'
	`); err != nil {
		t.Fatalf("failed to seed reserved inventory: %v", err)
	}
	if _, err := db.Exec(`
		UPDATE orders
		SET created_at = NOW() - INTERVAL '31 minutes'
		WHERE id = 'order-1'
	`); err != nil {
		t.Fatalf("failed to make order expired: %v", err)
	}

	expiredCount, err := repo.ExpireAcceptedOrders(time.Now().Add(-30 * time.Minute))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if expiredCount != 1 {
		t.Fatalf("expected 1 expired order, got %d", expiredCount)
	}

	var status string
	if err := db.QueryRow(`SELECT status FROM orders WHERE id = $1`, "order-1").Scan(&status); err != nil {
		t.Fatalf("failed to load status: %v", err)
	}
	if status != string(domain.OrderStatusCanceled) {
		t.Fatalf("expected canceled, got %s", status)
	}

	var reservationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM inventory_reservations WHERE order_id = $1`, "order-1").Scan(&reservationCount); err != nil {
		t.Fatalf("failed to count reservations: %v", err)
	}
	if reservationCount != 0 {
		t.Fatalf("expected no reservations after expiration, got %d", reservationCount)
	}

	var reserved int
	if err := db.QueryRow(`SELECT reserved FROM inventory WHERE sku = $1`, "sku-1").Scan(&reserved); err != nil {
		t.Fatalf("failed to load inventory reserved: %v", err)
	}
	if reserved != 0 {
		t.Fatalf("expected reserved to be released to 0, got %d", reserved)
	}
}

func TestOrderRepository_期限切れ_confirmedは対象外で変更しない(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	repo := NewOrderRepository(db)
	if _, err := db.Exec(`
		INSERT INTO inventory (sku, quantity, on_hand, reserved)
		VALUES ('sku-1', 100, 100, 0)
	`); err != nil {
		t.Fatalf("failed to seed inventory row: %v", err)
	}
	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 2}})
	if err := repo.Create(order, testQuotedUnitPrices(order.Items)); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, err := db.Exec(`
		UPDATE orders
		SET status = $1, created_at = NOW() - INTERVAL '31 minutes'
		WHERE id = $2
	`, domain.OrderStatusConfirmed, "order-1"); err != nil {
		t.Fatalf("failed to set confirmed order: %v", err)
	}
	if _, err := db.Exec(`
		UPDATE inventory
		SET reserved = 2, quantity = on_hand - 2
		WHERE sku = 'sku-1'
	`); err != nil {
		t.Fatalf("failed to seed reserved inventory: %v", err)
	}

	expiredCount, err := repo.ExpireAcceptedOrders(time.Now().Add(-30 * time.Minute))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if expiredCount != 0 {
		t.Fatalf("expected 0 expired orders, got %d", expiredCount)
	}

	var status string
	if err := db.QueryRow(`SELECT status FROM orders WHERE id = $1`, "order-1").Scan(&status); err != nil {
		t.Fatalf("failed to load status: %v", err)
	}
	if status != string(domain.OrderStatusConfirmed) {
		t.Fatalf("expected confirmed to remain unchanged, got %s", status)
	}

	var reservationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM inventory_reservations WHERE order_id = $1`, "order-1").Scan(&reservationCount); err != nil {
		t.Fatalf("failed to count reservations: %v", err)
	}
	if reservationCount != 1 {
		t.Fatalf("expected reservation rows to remain 1, got %d", reservationCount)
	}
}

func TestOrderRepository_期限切れ_期限未到達は対象外で変更しない(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ensureSchema(t, db)
	resetTables(t, db)

	repo := NewOrderRepository(db)
	if _, err := db.Exec(`
		INSERT INTO inventory (sku, quantity, on_hand, reserved)
		VALUES ('sku-1', 100, 100, 0)
	`); err != nil {
		t.Fatalf("failed to seed inventory row: %v", err)
	}
	order, _ := domain.NewOrder("order-1", "c-1", []domain.OrderItem{{SKU: "sku-1", Quantity: 2}})
	if err := repo.Create(order, testQuotedUnitPrices(order.Items)); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, err := db.Exec(`
		UPDATE orders
		SET created_at = NOW() - INTERVAL '10 minutes'
		WHERE id = $1
	`, "order-1"); err != nil {
		t.Fatalf("failed to set order age: %v", err)
	}
	if _, err := db.Exec(`
		UPDATE inventory
		SET reserved = 2, quantity = on_hand - 2
		WHERE sku = 'sku-1'
	`); err != nil {
		t.Fatalf("failed to seed reserved inventory: %v", err)
	}

	expiredCount, err := repo.ExpireAcceptedOrders(time.Now().Add(-30 * time.Minute))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if expiredCount != 0 {
		t.Fatalf("expected 0 expired orders, got %d", expiredCount)
	}

	var status string
	if err := db.QueryRow(`SELECT status FROM orders WHERE id = $1`, "order-1").Scan(&status); err != nil {
		t.Fatalf("failed to load status: %v", err)
	}
	if status != string(domain.OrderStatusAccepted) {
		t.Fatalf("expected accepted to remain unchanged, got %s", status)
	}

	var reservationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM inventory_reservations WHERE order_id = $1`, "order-1").Scan(&reservationCount); err != nil {
		t.Fatalf("failed to count reservations: %v", err)
	}
	if reservationCount != 1 {
		t.Fatalf("expected reservation rows to remain 1, got %d", reservationCount)
	}
}
