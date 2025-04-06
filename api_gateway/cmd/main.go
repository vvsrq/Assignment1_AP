package main

import (
	"api_gateway/config"
	"api_gateway/internal/middleware"
	"api_gateway/internal/proxy"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"log"
	"net/http"
	"os"
	"strconv"
)

func main() {

	cfg := config.LoadConfig()
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logLevel, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.Info("Starting API Gateway...")
	logger.Infof("Inventory Service target: %s", cfg.InventoryServiceURL)
	logger.Infof("Order Service target: %s", cfg.OrderServiceURL)

	inventoryProxy, err := proxy.NewReverseProxy(cfg.InventoryServiceURL, "/inventory", logger)
	if err != nil {
		log.Fatalf("FATAL: Failed to create inventory service proxy: %v", err)
	}
	orderProxy, err := proxy.NewReverseProxy(cfg.OrderServiceURL, "", logger) // Передаем пустой prefixToStrip
	if err != nil {
		log.Fatalf("FATAL: Failed to create order service proxy: %v", err)
	}

	router := gin.Default()
	router.RedirectTrailingSlash = false
	router.Use(gin.Recovery())

	router.GET("/", func(c *gin.Context) {
		// c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlDocs))
	})

	router.GET("/generate-test-token/:userID", func(c *gin.Context) {

		handlerLogger := logger.WithField("handler", "generate-test-token")
		handlerLogger.Info("Request received")

		userIDStr := c.Param("userID")
		userID, err := strconv.Atoi(userIDStr)
		if err != nil || userID <= 0 {
			handlerLogger.Warnf("Invalid user ID format received: %s", userIDStr)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
			return
		}
		handlerLogger.Infof("Parsed UserID: %d", userID)

		handlerLogger.Info("Attempting to generate token...")
		token, err := middleware.GenerateTestToken(userID, cfg.JWTSecret)
		if err != nil {
			handlerLogger.Errorf("Failed to generate test token: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		handlerLogger.Infof("Token generated successfully. Value (first 10 chars): %s...", token[:min(10, len(token))])

		responseMap := gin.H{"token": token}
		handlerLogger.Debugf("Prepared response map: %+v", responseMap)

		responseBytes, jsonErr := json.Marshal(responseMap)
		if jsonErr != nil {

			handlerLogger.Errorf("!!! Failed to manually marshal response map: %v", jsonErr)

			c.Status(http.StatusInternalServerError)
			return
		} else {
			handlerLogger.Debugf("Manual JSON marshal result: %s", string(responseBytes))
		}

		handlerLogger.Info("Sending token response...")
		c.JSON(http.StatusOK, responseMap)
		handlerLogger.Info("Token response supposedly sent.")

	})

	router.GET("/health", func(c *gin.Context) {
		logger.WithField("handler", "health").Info("Health check requested")
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	protected := router.Group("/")
	protected.Use(middleware.JWTMiddleware(cfg.JWTSecret, logger))
	{

		inventoryGroup := protected.Group("/inventory")
		{
			inventoryGroup.POST("/categories", proxy.ProxyHandler(inventoryProxy, logger))
			inventoryGroup.GET("/categories", proxy.ProxyHandler(inventoryProxy, logger))
			inventoryGroup.GET("/categories/:id", proxy.ProxyHandler(inventoryProxy, logger))
			inventoryGroup.PATCH("/categories/:id", proxy.ProxyHandler(inventoryProxy, logger))
			inventoryGroup.DELETE("/categories/:id", proxy.ProxyHandler(inventoryProxy, logger))

			inventoryGroup.POST("/products", proxy.ProxyHandler(inventoryProxy, logger))
			inventoryGroup.GET("/products", proxy.ProxyHandler(inventoryProxy, logger))
			inventoryGroup.GET("/products/:id", proxy.ProxyHandler(inventoryProxy, logger))
			inventoryGroup.PATCH("/products/:id", proxy.ProxyHandler(inventoryProxy, logger))
			inventoryGroup.DELETE("/products/:id", proxy.ProxyHandler(inventoryProxy, logger))
		}

		orderGroup := protected.Group("/orders")
		{
			orderGroup.POST("", proxy.ProxyHandler(orderProxy, logger))
			orderGroup.GET("", proxy.ProxyHandler(orderProxy, logger))
			orderGroup.Any("/*proxyPath", proxy.ProxyHandler(orderProxy, logger))
		}
	}

	// --- Start Server ---
	logger.Infof("API Gateway listening on port %s", cfg.GatewayPort)
	if err := router.Run(cfg.GatewayPort); err != nil {
		logger.Errorf("Failed to start API Gateway: %v", err)
		os.Exit(1)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
