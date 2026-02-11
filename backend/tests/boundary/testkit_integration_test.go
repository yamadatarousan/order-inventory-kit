package boundary_test

import (
	"database/sql"
	"os"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"order-inventory-kit/internal/adapter/handler"
	dbinfra "order-inventory-kit/internal/infra/db"
	"order-inventory-kit/internal/usecase"
)

type 境界統合Testkit struct {
	DB     *sql.DB
	Router *gin.Engine
}

func new境界統合Testkit(t *testing.T) *境界統合Testkit {
	t.Helper()

	db, ok := open境界統合DB(t)
	if !ok {
		t.Skip("skip integration test: database is not available")
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	ensure境界統合Schema(t, db)
	reset境界統合Tables(t, db)

	orderRepo := dbinfra.NewOrderRepository(db)
	paymentRepo := dbinfra.NewPaymentRepository(db)

	nextID := 0
	idGen := func() string {
		nextID++
		return "integration-order-" + strconv.Itoa(nextID)
	}
	uc := usecase.NewOrderUsecase(orderRepo, paymentRepo, idGen)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := handler.NewOrderHandler(uc)
	h.RegisterRoutes(router)

	return &境界統合Testkit{
		DB:     db,
		Router: router,
	}
}

func open境界統合DB(t *testing.T) (*sql.DB, bool) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		dsn = "postgres://order_inventory:order_inventory@localhost:5434/order_inventory_test?sslmode=disable"
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, false
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, false
	}
	return db, true
}

func ensure境界統合Schema(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS orders (
		  id TEXT PRIMARY KEY,
		  customer_id TEXT,
		  status TEXT NOT NULL,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS order_items (
		  order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
		  sku TEXT NOT NULL,
		  quantity INTEGER NOT NULL CHECK (quantity >= 1),
		  PRIMARY KEY (order_id, sku)
		);
		CREATE TABLE IF NOT EXISTS payments (
		  order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
		  idempotency_key TEXT NOT NULL,
		  status TEXT NOT NULL,
		  PRIMARY KEY (order_id, idempotency_key)
		);
		CREATE TABLE IF NOT EXISTS inventory (
		  sku TEXT PRIMARY KEY,
		  quantity INTEGER NOT NULL CHECK (quantity >= 0)
		);
		ALTER TABLE orders
		  ADD COLUMN IF NOT EXISTS customer_id TEXT;
		UPDATE orders SET customer_id = 'unknown' WHERE customer_id IS NULL;
		ALTER TABLE orders
		  ALTER COLUMN customer_id SET NOT NULL;
	`)
	if err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}
}

func reset境界統合Tables(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(`
		TRUNCATE TABLE payments, order_items, orders, inventory RESTART IDENTITY CASCADE;
	`)
	if err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}
}
