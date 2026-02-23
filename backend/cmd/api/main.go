package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	defaultCORSAllowOrigin         = "http://localhost:5173"
	defaultOrderExpirationTTL      = 30 * time.Minute
	defaultOrderExpirationInterval = 1 * time.Minute
)

type appConfig struct {
	APIAddr                 string
	DatabaseURL             string
	CORSAllowOrigin         string
	OrderExpirationTTL      time.Duration
	OrderExpirationInterval time.Duration
}

func loadConfig() appConfig {
	return appConfig{
		APIAddr:                 getenv("API_ADDR", defaultAPIAddr),
		DatabaseURL:             getenv("DATABASE_URL", defaultDatabaseURL),
		CORSAllowOrigin:         getenv("CORS_ALLOW_ORIGIN", defaultCORSAllowOrigin),
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

func newRouter(uc *usecase.OrderUsecase, corsAllowOrigin string) *gin.Engine {
	r := gin.New()
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Printf("failed to set trusted proxies: %v", err)
	}
	r.Use(corsMiddleware(corsAllowOrigin))
	r.Use(gin.Recovery())
	h := handler.NewOrderHandler(uc)
	h.RegisterRoutes(r)
	return r
}

// corsMiddleware はフロントエンド開発サーバーからの呼び出しを許可する。
func corsMiddleware(allowOrigin string) gin.HandlerFunc {
	allowed := parseAllowedOrigins(allowOrigin)
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && isAllowedOrigin(origin, allowed) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
		}
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func parseAllowedOrigins(raw string) map[string]struct{} {
	allowed := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		origin := strings.TrimSpace(part)
		if origin == "" {
			continue
		}
		allowed[origin] = struct{}{}
	}
	return allowed
}

func isAllowedOrigin(origin string, allowed map[string]struct{}) bool {
	if len(allowed) == 0 {
		return false
	}
	if _, ok := allowed["*"]; ok {
		return true
	}
	_, ok := allowed[origin]
	return ok
}

func migrationFiles() ([]string, error) {
	patterns := []string{
		"migrations/*.up.sql",
		"backend/migrations/*.up.sql",
		"../../migrations/*.up.sql",
	}

	fileSet := make(map[string]struct{})
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("glob migration files (%s): %w", pattern, err)
		}
		for _, path := range matches {
			fileSet[path] = struct{}{}
		}
	}

	files := make([]string, 0, len(fileSet))
	for path := range fileSet {
		files = append(files, path)
	}
	sort.Strings(files)
	return files, nil
}

func applyMigrations(db *sql.DB) error {
	files, err := migrationFiles()
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return errors.New("migration files not found")
	}

	for _, path := range files {
		sqlBytes, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read migration file (%s): %w", path, readErr)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, execErr := db.ExecContext(ctx, string(sqlBytes))
		cancel()
		if execErr != nil {
			return fmt.Errorf("apply migration (%s): %w", path, execErr)
		}
	}

	return nil
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
	if err := applyMigrations(db); err != nil {
		log.Fatalf("failed to apply migrations: %v", err)
	}

	orderRepo := infra.NewOrderRepository(db)
	paymentRepo := infra.NewPaymentRepository(db)
	inventoryRepo := infra.NewInventoryRepository(db)
	customerRepo := infra.NewCustomerRepository(db)
	uc := usecase.NewOrderUsecase(orderRepo, paymentRepo, inventoryRepo, customerRepo, newOrderID)
	router := newRouter(uc, cfg.CORSAllowOrigin)
	startOrderExpirationWorker(orderRepo, cfg.OrderExpirationInterval, cfg.OrderExpirationTTL)

	log.Printf("api server starting: addr=%s", cfg.APIAddr)
	if err := router.Run(cfg.APIAddr); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
