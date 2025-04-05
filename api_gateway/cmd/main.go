package main

import (
	"api_gateway/config"
	"api_gateway/internal/middleware"
	"api_gateway/internal/proxy"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"log"
	"os"
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
	orderProxy, err := proxy.NewReverseProxy(cfg.OrderServiceURL, "/orders", logger)
	if err != nil {
		log.Fatalf("FATAL: Failed to create order service proxy: %v", err)
	}

	router := gin.Default()
	router.RedirectTrailingSlash = false
	router.Use(gin.Recovery())

	router.GET("/", func(c *gin.Context) {
	})

	router.GET("/generate-test-token/:userID", func(c *gin.Context) { /* ... */ })
	router.GET("/health", func(c *gin.Context) { /* ... */ })

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
