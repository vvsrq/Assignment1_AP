package delivery

import (
	"inventory_service/internal/domain"
	"inventory_service/internal/usecase"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type ProductHandler struct {
	useCase usecase.ProductUseCase
	log     *logrus.Logger
}

func NewProductHandler(uc usecase.ProductUseCase, logger *logrus.Logger) *ProductHandler {
	return &ProductHandler{
		useCase: uc,
		log:     logger,
	}
}

func (h *ProductHandler) RegisterRoutes(router gin.IRouter) {
	products := router.Group("/products")
	{
		products.POST("", h.CreateProduct)
		products.GET("", h.ListProducts)
		products.GET("/:id", h.GetProductByID)
		products.PATCH("/:id", h.UpdateProduct)
		products.DELETE("/:id", h.DeleteProduct)
	}
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var product domain.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		h.log.Errorf("Failed to bind JSON for create product: %v", err)
		ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	createdProduct, err := h.useCase.CreateProduct(&product)
	if err != nil {
		statusCode := mapErrorToStatus(err)
		h.log.Errorf("Failed to create product '%s': %v", product.Name, err)
		ErrorResponse(c, statusCode, "Failed to create product: "+err.Error())
		return
	}

	h.log.Infof("Product created successfully: ID %d, Name %s", createdProduct.ID, createdProduct.Name)
	SuccessResponse(c, http.StatusCreated, "Product created successfully", createdProduct)
}

func (h *ProductHandler) GetProductByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		h.log.Warnf("Invalid product ID parameter: %s", idStr)
		ErrorResponse(c, http.StatusBadRequest, "Invalid product ID format")
		return
	}

	product, err := h.useCase.GetProductByID(id)
	if err != nil {
		statusCode := mapErrorToStatus(err) // Will map "not found" to 404
		h.log.Warnf("Failed to get product by ID %d: %v", id, err)
		ErrorResponse(c, statusCode, "Failed to retrieve product: "+err.Error())
		return
	}

	h.log.Infof("Product retrieved successfully: ID %d", id)
	SuccessResponse(c, http.StatusOK, "Product retrieved successfully", product)
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		h.log.Warnf("Invalid product ID parameter for update: %s", idStr)
		ErrorResponse(c, http.StatusBadRequest, "Invalid product ID format")
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		h.log.Errorf("Failed to bind JSON for update product ID %d: %v", id, err)
		ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if len(updates) == 0 {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request body: no fields provided for update")
		return
	}

	updatedProduct, err := h.useCase.UpdateProduct(id, updates) // Передаем ID и map
	if err != nil {
		statusCode := mapErrorToStatus(err)
		h.log.Errorf("Failed to update product ID %d: %v", id, err)
		ErrorResponse(c, statusCode, "Failed to update product: "+err.Error())
		return
	}

	h.log.Infof("Product updated successfully: ID %d", updatedProduct.ID)
	SuccessResponse(c, http.StatusOK, "Product updated successfully", updatedProduct)
}

func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		h.log.Warnf("Invalid product ID parameter for delete: %s", idStr)
		ErrorResponse(c, http.StatusBadRequest, "Invalid product ID format")
		return
	}

	err = h.useCase.DeleteProduct(id)
	if err != nil {
		statusCode := mapErrorToStatus(err)
		h.log.Warnf("Failed to delete product ID %d: %v", id, err)
		ErrorResponse(c, statusCode, "Failed to delete product: "+err.Error())
		return
	}

	h.log.Infof("Product deleted successfully: ID %d", id)
	SuccessResponse(c, http.StatusOK, "Product deleted successfully", nil)
}

func (h *ProductHandler) ListProducts(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")
	categoryIDStr := c.Query("category_id") // Optional filter

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		h.log.Warnf("Invalid limit parameter '%s', using default 10", limitStr)
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		h.log.Warnf("Invalid offset parameter '%s', using default 0", offsetStr)
		offset = 0
	}

	var products []domain.Product
	var listErr error

	if categoryIDStr != "" {
		categoryID, err := strconv.Atoi(categoryIDStr)
		if err != nil || categoryID <= 0 {
			h.log.Warnf("Invalid category_id filter parameter: %s", categoryIDStr)
			ErrorResponse(c, http.StatusBadRequest, "Invalid category_id format")
			return
		}
		h.log.Infof("Listing products for category %d with limit %d, offset %d", categoryID, limit, offset)
		products, listErr = h.useCase.ListProductsByCategory(categoryID, limit, offset)
	} else {
		h.log.Infof("Listing all products with limit %d, offset %d", limit, offset)
		products, listErr = h.useCase.ListProducts(limit, offset)
	}

	if listErr != nil {
		statusCode := mapErrorToStatus(listErr) // Check if it was a "category not found" error
		h.log.Errorf("Failed to list products: %v", listErr)
		ErrorResponse(c, statusCode, "Failed to retrieve products: "+listErr.Error())
		return
	}

	h.log.Infof("Retrieved %d products", len(products))
	if len(products) == 0 {
		SuccessResponse(c, http.StatusOK, "No products found matching criteria", []domain.Product{})
		return
	}
	SuccessResponse(c, http.StatusOK, "Products retrieved successfully", products)
}
