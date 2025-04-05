package main

import (
	"inventory_service/config"
	"inventory_service/internal/delivery"
	"inventory_service/internal/repository"
	"inventory_service/internal/usecase"
	"inventory_service/pkg/db"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"log"
)

// HTML content for the test page
const htmlTestPageContent = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Inventory Service API Test Page</title>
    <style>
        body { font-family: Helvetica, Arial, sans-serif; line-height: 1.6; padding: 20px; background-color: #f9f9f9; color: #333; }
        h1, h2 { border-bottom: 1px solid #ccc; padding-bottom: 5px; }
        ul { list-style: none; padding-left: 0; }
        li { margin-bottom: 15px; background-color: #fff; padding: 10px; border: 1px solid #eee; border-radius: 4px; }
        code { background-color: #e8e8e8; padding: 3px 6px; border-radius: 3px; font-family: Consolas, Monaco, monospace; }
        .method { font-weight: bold; display: inline-block; width: 60px; }
        .method-post { color: #49cc90; }
        .method-get { color: #61affe; }
        .method-patch { color: #fca130; }
        .method-delete { color: #f93e3e; }
        a { color: #007bff; text-decoration: none; }
        a:hover { text-decoration: underline; }
        p > code { font-size: 0.9em; }
    </style>
</head>
<body>
    <h1>Inventory Service API Endpoints</h1>
    <p>Base URL: <code>http://localhost:8081</code></p>

    <h2>Categories API</h2>
    <ul>
        <li><span class="method method-post">POST</span> <code>/categories</code> - Create a new category. Requires JSON body: <code>{"name": "string"}</code></li>
        <li><span class="method method-get">GET</span> <code><a href="/categories">/categories</a></code> - List all available categories.</li>
        <li><span class="method method-get">GET</span> <code>/categories/{id}</code> - Retrieve a specific category by its ID (e.g., <a href="/categories/1">/categories/1</a>).</li>
        <li><span class="method method-patch">PATCH</span> <code>/categories/{id}</code> - Update a category's name by its ID. Requires JSON body: <code>{"name": "string"}</code></li>
        <li><span class="method method-delete">DELETE</span> <code>/categories/{id}</code> - Delete a category by its ID.</li>
    </ul>

    <h2>Products API</h2>
    <ul>
        <li><span class="method method-post">POST</span> <code>/products</code> - Create a new product. Requires JSON body: <code>{"name": "string", "price": float64, "stock": int, "category_id": int}</code> (<code>category_id: 0</code> means no category).</li>
        <li><span class="method method-get">GET</span> <code><a href="/products">/products</a></code> - List products. Supports query parameters: <code>limit</code> (int, default 10), <code>offset</code> (int, default 0), <code>category_id</code> (int, filters by category). (e.g., <a href="/products?limit=5&offset=0">/products?limit=5&offset=0</a>, <a href="/products?category_id=1">/products?category_id=1</a>)</li>
        <li><span class="method method-get">GET</span> <code>/products/{id}</code> - Retrieve a specific product by its ID (e.g., <a href="/products/1">/products/1</a>).</li>
        <li><span class="method method-patch">PATCH</span> <code>/products/{id}</code> - Update a product by its ID. JSON body can contain any fields to update (<code>name</code>, <code>price</code>, <code>stock</code>, <code>category_id</code>).</li>
        <li><span class="method method-delete">DELETE</span> <code>/products/{id}</code> - Delete a product by its ID.</li>
    </ul>

</body>
</html>
`

func serveTestPage(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlTestPageContent))
}

func main() {
	//  Configuration and Logging Setup
	_ = config.LoadConfig()

	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})

	logger.Info("Starting Inventory Service...")

	// --- Database Connection ---
	database, err := db.Connect()
	if err != nil {
		log.Fatalf("FATAL: Failed to connect to database: %v", err)
	}
	defer database.Close()
	logger.Info("Database connection established.")

	// --- Dependency Injection ---
	// Repository Layer
	categoryRepo := repository.NewPostgresCategoryRepository(database, logger)
	productRepo := repository.NewPostgresProductRepository(database, logger)
	logger.Info("Repositories initialized.")

	// Usecase Layer
	categoryUseCase := usecase.NewCategoryUseCase(categoryRepo, logger)
	productUseCase := usecase.NewProductUseCase(productRepo, categoryRepo, logger)
	logger.Info("Use cases initialized.")

	categoryHandler := delivery.NewCategoryHandler(categoryUseCase, logger)
	productHandler := delivery.NewProductHandler(productUseCase, logger)
	logger.Info("Handlers initialized.")

	router := gin.Default()

	router.Use(gin.Recovery())

	router.Use(func(c *gin.Context) {
		logger.WithFields(logrus.Fields{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
			"ip":     c.ClientIP(),
		}).Info("Request received")
		c.Next()
		logger.WithFields(logrus.Fields{
			"status": c.Writer.Status(),
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
		}).Info("Request completed")
	})

	//Route Registration

	router.GET("/", serveTestPage)
	logger.Info("Registered HTML test page route at /")

	categoryHandler.RegisterRoutes(router)
	productHandler.RegisterRoutes(router)
	logger.Info("API Routes registered.")

	//  Start Server
	port := ":8081"
	logger.Infof("Starting server on port %s", port)
	if err := router.Run(port); err != nil {
		logger.Fatalf("Failed to start server: %v", err)
	}
}
