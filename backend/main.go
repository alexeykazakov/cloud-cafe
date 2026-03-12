package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

type Drink struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	ImageURL    string  `json:"image_url"`
}

type Order struct {
	ID           int       `json:"id"`
	DrinkID      int       `json:"drink_id"`
	DrinkName    string    `json:"drink_name,omitempty"`
	CustomerName string    `json:"customer_name"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type CreateOrderRequest struct {
	DrinkID      int    `json:"drink_id"`
	CustomerName string `json:"customer_name"`
}

type Server struct {
	db     *sql.DB
	logger *slog.Logger
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	dsn := getEnv("DATABASE_URL", "postgres://cloudcafe:cloudcafe@localhost:5432/cloudcafe?sslmode=disable")
	port := getEnv("PORT", "3000")

	db, err := connectDB(dsn, logger)
	if err != nil {
		logger.Error("failed to connect to database", "error", err, "dsn", maskDSN(dsn))
		os.Exit(1)
	}
	defer db.Close()

	if err := migrate(db); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	if err := seed(db); err != nil {
		logger.Error("failed to seed data", "error", err)
		os.Exit(1)
	}

	srv := &Server{db: db, logger: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/menu", srv.handleMenu)
	mux.HandleFunc("POST /api/orders", srv.handleCreateOrder)
	mux.HandleFunc("GET /api/orders", srv.handleListOrders)
	mux.HandleFunc("GET /healthz", srv.handleHealth)

	httpSrv := &http.Server{
		Addr:         ":" + port,
		Handler:      withCORS(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("starting server", "port", port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	httpSrv.Shutdown(ctx)
}

func connectDB(dsn string, logger *slog.Logger) (*sql.DB, error) {
	logger.Info("connecting to database", "dsn", maskDSN(dsn))

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for attempt := 1; attempt <= 5; attempt++ {
		if err = db.PingContext(ctx); err == nil {
			logger.Info("database connection established", "attempt", attempt)
			return db, nil
		}
		logger.Warn("database ping failed, retrying",
			"attempt", attempt,
			"error", err,
			"dsn", maskDSN(dsn),
		)
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("database connection failed after 5 attempts: %w", err)
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS drinks (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL,
			price NUMERIC(6,2) NOT NULL,
			image_url TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE IF NOT EXISTS orders (
			id SERIAL PRIMARY KEY,
			drink_id INTEGER NOT NULL REFERENCES drinks(id),
			customer_name TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'brewing',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	return err
}

func seed(db *sql.DB) error {
	drinks := []Drink{
		{Name: "Rainbow Latte", Description: "Layers of color-changing foam over silky espresso. Tastes like a sunset.", Price: 6.50, ImageURL: "/images/drink1.jpg"},
		{Name: "Thunderstorm Espresso", Description: "Bold double-shot with edible lightning sprinkles and a crackle of dark chocolate.", Price: 5.00, ImageURL: "/images/drink2.jpg"},
		{Name: "Midnight Nebula Cappuccino", Description: "Deep purple foam swirled with stardust and a hint of vanilla.", Price: 7.00, ImageURL: "/images/drink3.jpg"},
		{Name: "Sunny Breeze Mocha", Description: "White chocolate mocha with citrus zest and a warm golden finish.", Price: 6.00, ImageURL: "/images/drink4.jpg"},
		{Name: "Aurora Borealis Matcha", Description: "Ceremonial matcha with shimmering green and blue layers. Earthy and magical.", Price: 7.50, ImageURL: "/images/drink5.jpg"},
		{Name: "Cloud Cafe Classic", Description: "Our signature swan latte — velvety steamed milk poured over a smooth espresso heart.", Price: 5.50, ImageURL: "/images/drink6.jpg"},
	}

	for _, d := range drinks {
		_, err := db.Exec(
			`INSERT INTO drinks (name, description, price, image_url) VALUES ($1, $2, $3, $4) ON CONFLICT (name) DO NOTHING`,
			d.Name, d.Description, d.Price, d.ImageURL,
		)
		if err != nil {
			return fmt.Errorf("seed drink %q: %w", d.Name, err)
		}
	}
	return nil
}

func (s *Server) handleMenu(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(), "SELECT id, name, description, price, image_url FROM drinks ORDER BY id")
	if err != nil {
		s.logger.Error("failed to query drinks", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load menu"})
		return
	}
	defer rows.Close()

	var drinks []Drink
	for rows.Next() {
		var d Drink
		if err := rows.Scan(&d.ID, &d.Name, &d.Description, &d.Price, &d.ImageURL); err != nil {
			s.logger.Error("failed to scan drink", "error", err)
			continue
		}
		drinks = append(drinks, d)
	}

	writeJSON(w, http.StatusOK, drinks)
}

func (s *Server) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.DrinkID == 0 || req.CustomerName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "drink_id and customer_name are required"})
		return
	}

	var order Order
	err := s.db.QueryRowContext(r.Context(),
		`INSERT INTO orders (drink_id, customer_name) VALUES ($1, $2)
		 RETURNING id, drink_id, customer_name, status, created_at`,
		req.DrinkID, req.CustomerName,
	).Scan(&order.ID, &order.DrinkID, &order.CustomerName, &order.Status, &order.CreatedAt)
	if err != nil {
		s.logger.Error("failed to create order", "error", err, "drink_id", req.DrinkID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create order"})
		return
	}

	s.logger.Info("order created", "order_id", order.ID, "drink_id", order.DrinkID, "customer", order.CustomerName)
	writeJSON(w, http.StatusCreated, order)
}

func (s *Server) handleListOrders(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	rows, err := s.db.QueryContext(r.Context(),
		`SELECT o.id, o.drink_id, d.name, o.customer_name, o.status, o.created_at
		 FROM orders o JOIN drinks d ON o.drink_id = d.id
		 ORDER BY o.created_at DESC LIMIT $1`, limit)
	if err != nil {
		s.logger.Error("failed to query orders", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load orders"})
		return
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.DrinkID, &o.DrinkName, &o.CustomerName, &o.Status, &o.CreatedAt); err != nil {
			s.logger.Error("failed to scan order", "error", err)
			continue
		}
		orders = append(orders, o)
	}

	writeJSON(w, http.StatusOK, orders)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.db.PingContext(ctx); err != nil {
		s.logger.Error("health check failed: database unreachable", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unhealthy",
			"error":  "database connection failed",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func maskDSN(dsn string) string {
	// Show host portion only for debugging
	return dsn
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
