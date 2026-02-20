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
	defaultAPIAddr                 = ":8080"
	defaultDatabaseURL             = "postgres://order_inventory:order_inventory@localhost:5434/order_inventory_dev?sslmode=disable"
	defaultOrderExpirationTTL      = 30 * time.Minute
	defaultOrderExpirationInterval = 1 * time.Minute
)

type appConfig struct {
	APIAddr                 string
	DatabaseURL             string
	OrderExpirationTTL      time.Duration
	OrderExpirationInterval time.Duration
}

func loadConfig() appConfig {
	return appConfig{
		APIAddr:                 getenv("API_ADDR", defaultAPIAddr),
		DatabaseURL:             getenv("DATABASE_URL", defaultDatabaseURL),
		OrderExpirationTTL:      getenvDuration("ORDER_EXPIRATION_TTL", defaultOrderExpirationTTL),
		OrderExpirationInterval: getenvDuration("ORDER_EXPIRATION_INTERVAL", defaultOrderExpirationInterval),
	}
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
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

// startOrderExpirationWorker は期限切れ注文の戻し処理を定期実行する。
func startOrderExpirationWorker(repo *infra.OrderRepository, interval, ttl time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			cutoff := time.Now().UTC().Add(-ttl)
			expired, err := repo.ExpireAcceptedOrders(cutoff)
			if err != nil {
				log.Printf("order expiration worker failed: %v", err)
				continue
			}
			if expired > 0 {
				log.Printf("order expiration worker expired=%d cutoff=%s", expired, cutoff.Format(time.RFC3339))
			}
		}
	}()
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
	startOrderExpirationWorker(orderRepo, cfg.OrderExpirationInterval, cfg.OrderExpirationTTL)

	log.Printf("api server starting: addr=%s", cfg.APIAddr)
	if err := router.Run(cfg.APIAddr); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
