package boundary_test

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

	prepare境界統合DB(t, db)

	orderRepo := dbinfra.NewOrderRepository(db)
	paymentRepo := dbinfra.NewPaymentRepository(db)
	inventoryRepo := dbinfra.NewInventoryRepository(db)

	idGen := new固定IDGenerator(
		"integration-order-1",
		"integration-order-2",
		"integration-order-3",
	)
	uc := usecase.NewOrderUsecase(orderRepo, paymentRepo, inventoryRepo, idGen)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := handler.NewOrderHandler(uc)
	h.RegisterRoutes(router)

	return &境界統合Testkit{
		DB:     db,
		Router: router,
	}
}

// new固定IDGenerator は非決定要素（ID）の扱いを固定する。
// 統合境界テストでは実行ごとの差分を避けるため、IDは固定列から順に払い出す。
func new固定IDGenerator(ids ...string) func() string {
	next := 0
	return func() string {
		if next >= len(ids) {
			panic(fmt.Sprintf("fixed id exhausted: need more than %d ids", len(ids)))
		}
		id := ids[next]
		next++
		return id
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

// prepare境界統合DB は統合境界テスト用のDB準備/後片付け手順を固定する。
// 手順: migrate適用 -> テーブルリセット -> seed投入 -> テスト終了時リセット。
func prepare境界統合DB(t *testing.T, db *sql.DB) {
	t.Helper()

	apply境界統合Migrations(t, db)
	reset境界統合Tables(t, db)
	seed境界統合DB(t, db)

	t.Cleanup(func() {
		reset境界統合Tables(t, db)
	})
}

func apply境界統合Migrations(t *testing.T, db *sql.DB) {
	t.Helper()

	paths, err := filepath.Glob("../../migrations/*.up.sql")
	if err != nil {
		t.Fatalf("failed to list migration files: %v", err)
	}
	if len(paths) == 0 {
		t.Fatalf("migration files not found: ../../migrations/*.up.sql")
	}
	sort.Strings(paths)

	for _, path := range paths {
		// seedマイグレーションは毎テストで再投入するため、初期適用では除外する。
		if strings.HasSuffix(path, "_seed_inventory.up.sql") {
			continue
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("failed to read migration file (%s): %v", path, readErr)
		}
		if _, execErr := db.Exec(string(content)); execErr != nil {
			t.Fatalf("failed to apply migration (%s): %v", path, execErr)
		}
	}
}

func seed境界統合DB(t *testing.T, db *sql.DB) {
	t.Helper()

	seedPath := "../../migrations/0002_seed_inventory.up.sql"
	content, err := os.ReadFile(seedPath)
	if err != nil {
		t.Fatalf("failed to read seed file (%s): %v", seedPath, err)
	}
	if _, err := db.Exec(string(content)); err != nil {
		t.Fatalf("failed to apply seed (%s): %v", seedPath, err)
	}
}

func reset境界統合Tables(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(`
		TRUNCATE TABLE payments, inventory_reservations, order_items, orders, inventory RESTART IDENTITY CASCADE;
	`)
	if err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM inventory`).Scan(&count); err != nil {
		t.Fatalf("failed to count inventory rows after reset: %v", err)
	}
	if count != 0 {
		t.Fatalf("reset must empty inventory table, got %d rows", count)
	}
}
