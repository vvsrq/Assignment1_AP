package delivery

import (
	"inventory_service/internal/domain"
	"inventory_service/internal/usecase"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type CategoryHandler struct {
	useCase usecase.CategoryUseCase
	log     *logrus.Logger
}

func NewCategoryHandler(uc usecase.CategoryUseCase, logger *logrus.Logger) *CategoryHandler {
	return &CategoryHandler{
		useCase: uc,
		log:     logger,
	}
}

func (h *CategoryHandler) RegisterRoutes(router gin.IRouter) {
	categories := router.Group("/categories")
	{
		categories.POST("", h.CreateCategory)
		categories.GET("", h.ListCategories)
		categories.GET("/:id", h.GetCategoryByID)
		categories.PATCH("/:id", h.UpdateCategory)
		categories.DELETE("/:id", h.DeleteCategory)
	}
}

func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var category domain.Category
	if err := c.ShouldBindJSON(&category); err != nil {
		h.log.Errorf("Failed to bind JSON for create category: %v", err)
		ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	createdCategory, err := h.useCase.CreateCategory(&category)
	if err != nil {
		statusCode := mapErrorToStatus(err)
		h.log.Errorf("Failed to create category '%s': %v", category.Name, err)
		ErrorResponse(c, statusCode, "Failed to create category: "+err.Error())
		return
	}

	h.log.Infof("Category created successfully: ID %d, Name %s", createdCategory.ID, createdCategory.Name)
	SuccessResponse(c, http.StatusCreated, "Category created successfully", createdCategory)
}

func (h *CategoryHandler) GetCategoryByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		h.log.Warnf("Invalid category ID parameter: %s", idStr)
		ErrorResponse(c, http.StatusBadRequest, "Invalid category ID format")
		return
	}

	category, err := h.useCase.GetCategoryByID(id)
	if err != nil {
		statusCode := mapErrorToStatus(err)
		h.log.Warnf("Failed to get category by ID %d: %v", id, err)
		ErrorResponse(c, statusCode, "Failed to retrieve category: "+err.Error())
		return
	}

	h.log.Infof("Category retrieved successfully: ID %d", id)
	SuccessResponse(c, http.StatusOK, "Category retrieved successfully", category)
}

func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		h.log.Warnf("Invalid category ID parameter for update: %s", idStr)
		ErrorResponse(c, http.StatusBadRequest, "Invalid category ID format")
		return
	}

	var categoryUpdates domain.Category

	if err := c.ShouldBindJSON(&categoryUpdates); err != nil {
		h.log.Errorf("Failed to bind JSON for update category ID %d: %v", id, err)
		ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	categoryUpdates.ID = id

	updatedCategory, err := h.useCase.UpdateCategory(&categoryUpdates)
	if err != nil {
		statusCode := mapErrorToStatus(err)
		h.log.Errorf("Failed to update category ID %d: %v", id, err)
		ErrorResponse(c, statusCode, "Failed to update category: "+err.Error())
		return
	}

	h.log.Infof("Category updated successfully: ID %d", updatedCategory.ID)
	SuccessResponse(c, http.StatusOK, "Category updated successfully", updatedCategory)
}

func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		h.log.Warnf("Invalid category ID parameter for delete: %s", idStr)
		ErrorResponse(c, http.StatusBadRequest, "Invalid category ID format")
		return
	}

	err = h.useCase.DeleteCategory(id)
	if err != nil {
		statusCode := mapErrorToStatus(err)
		h.log.Warnf("Failed to delete category ID %d: %v", id, err)
		ErrorResponse(c, statusCode, "Failed to delete category: "+err.Error())
		return
	}

	h.log.Infof("Category deleted successfully: ID %d", id)
	SuccessResponse(c, http.StatusOK, "Category deleted successfully", nil) // No data to return on successful delete
}

func (h *CategoryHandler) ListCategories(c *gin.Context) {
	categories, err := h.useCase.ListCategories()
	if err != nil {
		h.log.Errorf("Failed to list categories: %v", err)
		ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve categories: "+err.Error())
		return
	}

	h.log.Infof("Retrieved %d categories", len(categories))
	if len(categories) == 0 {
		// Return success with empty array instead of just null data
		SuccessResponse(c, http.StatusOK, "No categories found", []domain.Category{})
		return
	}

	SuccessResponse(c, http.StatusOK, "Categories retrieved successfully", categories)
}
