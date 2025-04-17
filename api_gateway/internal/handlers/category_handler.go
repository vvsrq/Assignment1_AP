package handlers

import (
	"api_gateway/internal/clients"
	inventorypb "api_gateway/proto/inventorypb"
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type CategoryHandler struct {
	inventoryClient clients.InventoryServiceClient
	log             *logrus.Logger
}

func NewCategoryHandler(ic clients.InventoryServiceClient, logger *logrus.Logger) *CategoryHandler {
	return &CategoryHandler{
		inventoryClient: ic,
		log:             logger,
	}
}

type CreateCategoryRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	handlerLogger := h.log.WithField("handler", "CreateCategory")
	var req CreateCategoryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		handlerLogger.Warnf("Failed to bind request: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body: " + err.Error()})
		return
	}

	grpcReq := &inventorypb.CreateCategoryRequest{Name: req.Name}

	ctxWithMD := getContextWithAuthToken(c)
	callCtx, cancel := context.WithTimeout(ctxWithMD, 5*time.Second)
	defer cancel()

	grpcRes, err := h.inventoryClient.CreateCategory(callCtx, grpcReq)
	if err != nil {
		mapGrpcErrorToHttpStatus(c, handlerLogger, err)
		return
	}

	c.JSON(http.StatusCreated, grpcRes)
}

func (h *CategoryHandler) GetCategory(c *gin.Context) {
	handlerLogger := h.log.WithField("handler", "GetCategory")
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		handlerLogger.Warnf("Invalid category ID parameter: %s", idStr)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid category ID format"})
		return
	}

	grpcReq := &inventorypb.GetCategoryRequest{Id: id}

	ctxWithMD := getContextWithAuthToken(c)
	callCtx, cancel := context.WithTimeout(ctxWithMD, 5*time.Second)
	defer cancel()

	grpcRes, err := h.inventoryClient.GetCategory(callCtx, grpcReq)
	if err != nil {
		mapGrpcErrorToHttpStatus(c, handlerLogger, err)
		return
	}

	c.JSON(http.StatusOK, grpcRes)
}

func (h *CategoryHandler) ListCategories(c *gin.Context) {
	handlerLogger := h.log.WithField("handler", "ListCategories")

	grpcReq := &inventorypb.ListCategoriesRequest{}

	ctxWithMD := getContextWithAuthToken(c)
	callCtx, cancel := context.WithTimeout(ctxWithMD, 5*time.Second)
	defer cancel()

	grpcRes, err := h.inventoryClient.ListCategories(callCtx, grpcReq)
	if err != nil {
		mapGrpcErrorToHttpStatus(c, handlerLogger, err)
		return
	}

	c.JSON(http.StatusOK, grpcRes)
}

// UpdateCategoryRequest for binding JSON
type UpdateCategoryRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	handlerLogger := h.log.WithField("handler", "UpdateCategory")
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		handlerLogger.Warnf("Invalid category ID parameter: %s", idStr)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid category ID format"})
		return
	}

	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerLogger.Warnf("Failed to bind request: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body: " + err.Error()})
		return
	}

	grpcReq := &inventorypb.UpdateCategoryRequest{
		Category: &inventorypb.Category{
			Id:   id,
			Name: req.Name,
		},
	}

	ctxWithMD := getContextWithAuthToken(c)
	callCtx, cancel := context.WithTimeout(ctxWithMD, 5*time.Second)
	defer cancel()

	grpcRes, err := h.inventoryClient.UpdateCategory(callCtx, grpcReq)
	if err != nil {
		mapGrpcErrorToHttpStatus(c, handlerLogger, err)
		return
	}

	c.JSON(http.StatusOK, grpcRes)
}

func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	handlerLogger := h.log.WithField("handler", "DeleteCategory")
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		handlerLogger.Warnf("Invalid category ID parameter: %s", idStr)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid category ID format"})
		return
	}

	grpcReq := &inventorypb.DeleteCategoryRequest{Id: id}

	ctxWithMD := getContextWithAuthToken(c)
	callCtx, cancel := context.WithTimeout(ctxWithMD, 5*time.Second)
	defer cancel()

	_, err = h.inventoryClient.DeleteCategory(callCtx, grpcReq)
	if err != nil {
		mapGrpcErrorToHttpStatus(c, handlerLogger, err)
		return
	}

	c.Status(http.StatusNoContent)
}
