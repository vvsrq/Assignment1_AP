package main

import (
	"api_gateway/config"
	"api_gateway/internal/clients"
	"api_gateway/internal/handlers"
	"api_gateway/internal/middleware"
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.JSONFormatter{})

	cfg := config.LoadConfig(logger)
	logLevel, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logLevel = logrus.InfoLevel
		logger.Warnf("Invalid log level '%s', using default 'info'. Error: %v", cfg.LogLevel, err)
	}
	logger.SetLevel(logLevel)
	logger.Infof("Starting API Gateway...")
	logger.Infof("Log level set to: %s", logLevel.String())

	clientTimeout := 5 * time.Second

	userClient, err := clients.NewUserServiceClient(cfg.UserServiceGrpcAddr, logger, clientTimeout)
	if err != nil {
		logger.Fatalf("FATAL: Failed to create User Service client: %v", err)
	}
	defer userClient.Close()

	inventoryClient, err := clients.NewInventoryServiceClient(cfg.InventoryServiceGrpcAddr, logger, clientTimeout)
	if err != nil {
		logger.Fatalf("FATAL: Failed to create Inventory Service client: %v", err)
	}
	defer inventoryClient.Close()

	orderClient, err := clients.NewOrderServiceClient(cfg.OrderServiceGrpcAddr, logger, clientTimeout)
	if err != nil {
		logger.Fatalf("FATAL: Failed to create Order Service client: %v", err)
	}
	defer orderClient.Close()

	logger.Info("gRPC Clients initialized successfully.")

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger(logger))

	authHandler := handlers.NewAuthHandler(userClient, logger)
	userHandler := handlers.NewUserHandler(userClient, logger)
	productHandler := handlers.NewProductHandler(inventoryClient, logger)
	categoryHandler := handlers.NewCategoryHandler(inventoryClient, logger)
	orderHandler := handlers.NewOrderHandler(orderClient, logger)
	logger.Info("HTTP Handlers initialized.")

	v1 := router.Group("/api/v1")

	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/login", authHandler.Login)
	}
	userGroupPublic := v1.Group("/users")
	{
		userGroupPublic.POST("/register", userHandler.Register)
	}

	// --- Protected Routes ---
	protected := v1.Group("/")

	protected.Use(middleware.AuthMiddleware(logger))
	{

		// --- Products ---
		products := protected.Group("/products")
		{
			products.POST("", productHandler.CreateProduct)
			products.GET("", productHandler.ListProducts)
			products.GET("/:id", productHandler.GetProduct)
			products.PATCH("/:id", productHandler.UpdateProduct)
			products.DELETE("/:id", productHandler.DeleteProduct)
		}

		// Categories
		categories := protected.Group("/categories")
		{
			categories.POST("", categoryHandler.CreateCategory)
			categories.GET("", categoryHandler.ListCategories)
			categories.GET("/:id", categoryHandler.GetCategory)
			categories.PATCH("/:id", categoryHandler.UpdateCategory)
			categories.DELETE("/:id", categoryHandler.DeleteCategory)
		}

		//  Orders
		orders := protected.Group("/orders")
		{
			orders.POST("", orderHandler.CreateOrder)
			orders.GET("", orderHandler.ListOrders)
			orders.GET("/:id", orderHandler.GetOrder)
			orders.PATCH("/:id", orderHandler.UpdateOrderStatus)
		}
		userGroupProtected := protected.Group("/users")
		{
			userGroupProtected.GET("/profile/:id", userHandler.GetProfile)
		}

	}

	// --- Health Check ---
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	//Start HTTP Server with Graceful Shutdown
	httpServer := &http.Server{
		Addr:    cfg.GatewayPort,
		Handler: router,
	}

	serverErrChan := make(chan error, 1)

	go func() {
		logger.Infof("API Gateway listening on %s", cfg.GatewayPort)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Errorf("Failed to start API Gateway server: %v", err)
			serverErrChan <- err
		}
		logger.Info("API Gateway HTTP server stopped serving.")
		close(serverErrChan)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	logger.Info("Signal listener started.")

	select {
	case sig := <-quit:
		logger.Warnf("Shutdown signal received: %v", sig)
	case err := <-serverErrChan:
		if err != nil {
			logger.Errorf("Server failed unexpectedly: %v", err)
		}
	}

	//Perform Graceful Shutdown
	logger.Info("Attempting graceful shutdown of HTTP server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Errorf("HTTP server graceful shutdown failed: %v", err)
	} else {
		logger.Info("HTTP server gracefully stopped.")
	}

	logger.Info("API Gateway shut down gracefully.")
}
