package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpAdapter "github.com/carlosindriago/agendadorplus/internal/adapters/http"
	"github.com/carlosindriago/agendadorplus/internal/adapters/notification"
	"github.com/carlosindriago/agendadorplus/internal/adapters/postgres"
	"github.com/carlosindriago/agendadorplus/internal/usecases"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// 1. Setup Logger (Structured JSON for production)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting AgendadorPlus backend...")

	// 2. Load Environment Variables (Using defaults if missing, panic on critical)
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Error("DATABASE_URL environment variable is required")
		os.Exit(1)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "change-me-to-a-real-secret-in-production"
		logger.Warn("Using default JWT_SECRET. This is insecure for production.")
	}

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080"
	}

	timeoutStr := os.Getenv("SERVER_TIMEOUT")
	serverTimeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		serverTimeout = 3 * time.Second
		logger.Warn("Using default server timeout of 3s", "provided", timeoutStr)
	}

	// 3. Initialize PostgreSQL Connection Pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		logger.Error("Failed to parse database config", "error", err)
		os.Exit(1)
	}
	
	// Pool settings
	config.MaxConns = 50
	config.MinConns = 10
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Error("Failed to ping database", "error", err)
		os.Exit(1)
	}
	logger.Info("Connected to PostgreSQL successfully")

	// 4. Dependency Injection Wiring (Manual)
	
	// Adapters - Driven
	dbRepo := postgres.NewRepository(pool, logger)
	notifier := notification.NewLogNotifier(logger)

	// Use Cases (implementing Driving Ports)
	bookingUC := usecases.NewBookingUseCase(dbRepo, notifier, logger)
	availabilityUC := usecases.NewAvailabilityUseCase(dbRepo, logger)
	generatorUC := usecases.NewSlotGeneratorUseCase(dbRepo, logger)
	authUC := usecases.NewAuthUseCase(dbRepo, jwtSecret, logger)

	// Adapters - Driving (HTTP)
	handlers := httpAdapter.NewHandlers(authUC, bookingUC, availabilityUC, generatorUC)
	router := httpAdapter.SetupRouter(handlers, jwtSecret, serverTimeout, "*") // Allow all origins for MVP

	// 5. Setup HTTP Server with Graceful Shutdown
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", serverPort),
		Handler: router,
		// Good practice timeouts
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("HTTP server starting", "port", serverPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("Server exiting")
}
