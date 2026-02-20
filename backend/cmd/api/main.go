package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"order-inventory-kit/internal/adapter/handler"
	infra "order-inventory-kit/internal/infra/db"
	"order-inventory-kit/internal/usecase"
)

const (
	defaultAPIAddr     = ":8080"
	defaultDatabaseURL = "postgres://order_inventory:order_inventory@localhost:5434/order_inventory_dev?sslmode=disable"
)

type appConfig struct {
	APIAddr     string
	DatabaseURL string
}

func loadConfig() appConfig {
	return appConfig{
		APIAddr:     getenv("API_ADDR", defaultAPIAddr),
		DatabaseURL: getenv("DATABASE_URL", defaultDatabaseURL),
	}
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func newOrderID() string {
	return time.Now().UTC().Format("20060102150405.000000000")
}

func newRouter(uc *usecase.OrderUsecase) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	h := handler.NewOrderHandler(uc)
	h.RegisterRoutes(r)
	return r
}

func main() {
	cfg := loadConfig()

	db, err := openDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()

	orderRepo := infra.NewOrderRepository(db)
	paymentRepo := infra.NewPaymentRepository(db)
	inventoryRepo := infra.NewInventoryRepository(db)
	uc := usecase.NewOrderUsecase(orderRepo, paymentRepo, inventoryRepo, newOrderID)
	router := newRouter(uc)

	log.Printf("api server starting: addr=%s", cfg.APIAddr)
	if err := router.Run(cfg.APIAddr); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
