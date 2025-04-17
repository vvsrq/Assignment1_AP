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
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type ProductHandler struct {
	inventoryClient clients.InventoryServiceClient
	log             *logrus.Logger
}

func NewProductHandler(ic clients.InventoryServiceClient, logger *logrus.Logger) *ProductHandler {
	return &ProductHandler{
		inventoryClient: ic,
		log:             logger,
	}
}

func getContextWithAuthToken(c *gin.Context) context.Context {
	ctx := c.Request.Context()
	if rawToken, exists := c.Get("rawToken"); exists {
		if tokenStr, ok := rawToken.(string); ok && tokenStr != "" {
			md := metadata.Pairs("x-auth-token", tokenStr)
			return metadata.NewOutgoingContext(ctx, md)
		}
	}
	return ctx
}

type CreateProductRequest struct {
	Name       string  `json:"name" binding:"required"`
	Price      float64 `json:"price" binding:"required,gt=0"`
	Stock      int32   `json:"stock" binding:"required,gte=0"`
	CategoryID int64   `json:"category_id"`
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	handlerLogger := h.log.WithField("handler", "CreateProduct")
	var req CreateProductRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		handlerLogger.Warnf("Failed to bind request: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body: " + err.Error()})
		return
	}

	grpcReq := &inventorypb.CreateProductRequest{
		Name:       req.Name,
		Price:      req.Price,
		Stock:      req.Stock,
		CategoryId: req.CategoryID,
	}

	ctxWithMD := getContextWithAuthToken(c)
	callCtx, cancel := context.WithTimeout(ctxWithMD, 5*time.Second)
	defer cancel()

	grpcRes, err := h.inventoryClient.CreateProduct(callCtx, grpcReq)
	if err != nil {
		mapGrpcErrorToHttpStatus(c, handlerLogger, err)
		return
	}

	c.JSON(http.StatusCreated, grpcRes)
}

func (h *ProductHandler) GetProduct(c *gin.Context) {
	handlerLogger := h.log.WithField("handler", "GetProduct")
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		handlerLogger.Warnf("Invalid product ID parameter: %s", idStr)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid product ID format"})
		return
	}

	grpcReq := &inventorypb.GetProductRequest{Id: id}

	ctxWithMD := getContextWithAuthToken(c)
	callCtx, cancel := context.WithTimeout(ctxWithMD, 5*time.Second)
	defer cancel()

	grpcRes, err := h.inventoryClient.GetProduct(callCtx, grpcReq)
	if err != nil {
		mapGrpcErrorToHttpStatus(c, handlerLogger, err)
		return
	}

	c.JSON(http.StatusOK, grpcRes)
}

func (h *ProductHandler) ListProducts(c *gin.Context) {
	handlerLogger := h.log.WithField("handler", "ListProducts")

	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")
	categoryIDStr := c.Query("category_id")

	limit, err := strconv.ParseInt(limitStr, 10, 32)
	if err != nil || limit < 0 {
		limit = 10 // Default
	}
	offset, err := strconv.ParseInt(offsetStr, 10, 32) // int32 for gRPC
	if err != nil || offset < 0 {
		offset = 0
	}

	grpcReq := &inventorypb.ListProductsRequest{
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	if categoryIDStr != "" {
		catID, err := strconv.ParseInt(categoryIDStr, 10, 64)
		if err != nil || catID <= 0 {
			handlerLogger.Warnf("Invalid category_id query parameter: %s", categoryIDStr)
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid category_id format"})
			return
		}
		grpcReq.CategoryIdFilter = &wrapperspb.Int64Value{Value: catID}
	}

	ctxWithMD := getContextWithAuthToken(c)
	callCtx, cancel := context.WithTimeout(ctxWithMD, 5*time.Second)
	defer cancel()

	grpcRes, err := h.inventoryClient.ListProducts(callCtx, grpcReq)
	if err != nil {
		mapGrpcErrorToHttpStatus(c, handlerLogger, err)
		return
	}
	c.JSON(http.StatusOK, grpcRes)
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	handlerLogger := h.log.WithField("handler", "UpdateProduct")
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		handlerLogger.Warnf("Invalid product ID parameter: %s", idStr)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid product ID format"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		handlerLogger.Warnf("Failed to bind update request body: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body: " + err.Error()})
		return
	}

	grpcProduct := &inventorypb.Product{Id: id}
	maskPaths := []string{}

	if name, ok := updates["name"].(string); ok {
		grpcProduct.Name = name
		maskPaths = append(maskPaths, "name")
	}
	if price, ok := updates["price"].(float64); ok {
		grpcProduct.Price = price
		maskPaths = append(maskPaths, "price")
	}
	if stockVal, ok := updates["stock"]; ok {
		var stockInt int32
		if stockFloat, okFloat := stockVal.(float64); okFloat {
			stockInt = int32(stockFloat)
		} else if stockIntVal, okInt := stockVal.(int); okInt {
			stockInt = int32(stockIntVal)
		} else if stockInt32Val, okInt32 := stockVal.(int32); okInt32 {
			stockInt = stockInt32Val
		} else {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid type for stock"})
			return
		}
		grpcProduct.Stock = stockInt
		maskPaths = append(maskPaths, "stock")
	}
	if catIDVal, ok := updates["category_id"]; ok {
		var catIDInt64 int64
		if catIDFloat, okFloat := catIDVal.(float64); okFloat {
			catIDInt64 = int64(catIDFloat)
		} else if catIDInt, okInt := catIDVal.(int); okInt {
			catIDInt64 = int64(catIDInt)
		} else if catIDInt64Val, okInt64 := catIDVal.(int64); okInt64 {
			catIDInt64 = catIDInt64Val
		} else if catIDVal == nil {
			catIDInt64 = 0
		} else {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid type for category_id"})
			return
		}
		grpcProduct.CategoryId = catIDInt64
		maskPaths = append(maskPaths, "category_id")
	}

	if len(maskPaths) == 0 {
		handlerLogger.Warn("Update request received, but no valid fields to update")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "No valid fields provided for update"})
		return
	}

	grpcReq := &inventorypb.UpdateProductRequest{
		Product:    grpcProduct,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: maskPaths},
	}

	ctxWithMD := getContextWithAuthToken(c)
	callCtx, cancel := context.WithTimeout(ctxWithMD, 5*time.Second)
	defer cancel()

	grpcRes, err := h.inventoryClient.UpdateProduct(callCtx, grpcReq)
	if err != nil {
		mapGrpcErrorToHttpStatus(c, handlerLogger, err)
		return
	}

	c.JSON(http.StatusOK, grpcRes)
}

func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	handlerLogger := h.log.WithField("handler", "DeleteProduct")
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		handlerLogger.Warnf("Invalid product ID parameter: %s", idStr)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid product ID format"})
		return
	}

	grpcReq := &inventorypb.DeleteProductRequest{Id: id}

	ctxWithMD := getContextWithAuthToken(c)
	callCtx, cancel := context.WithTimeout(ctxWithMD, 5*time.Second)
	defer cancel()

	_, err = h.inventoryClient.DeleteProduct(callCtx, grpcReq)
	if err != nil {
		mapGrpcErrorToHttpStatus(c, handlerLogger, err)
		return
	}

	c.Status(http.StatusNoContent)
}
