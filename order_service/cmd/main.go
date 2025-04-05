package main

import (
	"order_service/config"
	"order_service/internal/clients"
	"order_service/internal/delivery"
	"order_service/internal/repository"
	"order_service/internal/usecase"
	"order_service/pkg/db"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	cfg := config.LoadConfig()
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logLevel, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logLevel = logrus.InfoLevel
		logger.Warnf("Invalid LOG_LEVEL '%s', using default: %s", cfg.LogLevel, logLevel.String())
	}
	logger.SetLevel(logLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.Info("Starting Order Service...")
	logger.Infof("Log level set to: %s", logLevel.String())

	if cfg.DatabaseURL == "" {
	}
	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
	}
	defer database.Close()
	logger.Info("Database connection established.")

	if cfg.InventoryServiceURL == "" {
		logger.Fatal("FATAL: Inventory Service URL is not configured. Set INVENTORY_SERVICE_URL.")
	}
	inventoryClient := clients.NewInventoryHTTPClient(
		cfg.InventoryServiceURL,
		5*time.Second,
		logger,
	)
	logger.Infof("Inventory Service Client initialized for target: %s", cfg.InventoryServiceURL)

	// --- Dependency Injection ---
	orderRepo := repository.NewPostgresOrderRepository(database, logger)
	logger.Info("Repositories initialized.")

	orderUseCase := usecase.NewOrderUseCase(orderRepo, inventoryClient, logger)
	logger.Info("Use cases initialized.")
	orderHandler := delivery.NewOrderHandler(orderUseCase, logger)
	logger.Info("Handlers initialized.")

	router := gin.Default()
	router.RedirectTrailingSlash = false
	router.Use(func(c *gin.Context) { /* TODO Middleware logging */ })

	orderHandler.RegisterRoutes(router)
	logger.Info("Routes registered.")

	// --- Start Server ---
	port := cfg.Port
	if port == "" || port == ":" {
	}
	logger.Infof("Starting server on port %s", port)
	if err := router.Run(port); err != nil {
		logger.Errorf("Failed to start server on port %s: %v", port, err)
		os.Exit(1)
	}
}
