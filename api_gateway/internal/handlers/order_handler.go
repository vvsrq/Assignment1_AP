package handlers

import (
	"api_gateway/internal/clients"
	orderpb "api_gateway/proto/orderpb"
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type OrderHandler struct {
	orderClient clients.OrderServiceClient

	log *logrus.Logger
}

func NewOrderHandler(oc clients.OrderServiceClient, logger *logrus.Logger) *OrderHandler {
	return &OrderHandler{
		orderClient: oc,
		log:         logger,
	}
}

type CreateOrderItemRequest struct {
	ProductID int64 `json:"product_id" binding:"required,gt=0"`
	Quantity  int32 `json:"quantity" binding:"required,gt=0"`
}

type CreateOrderRequest struct {
	Items []CreateOrderItemRequest `json:"items" binding:"required,min=1,dive"`
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	handlerLogger := h.log.WithField("handler", "CreateOrder")
	var req CreateOrderRequest

	rawToken, _ := c.Get("rawToken")
	if rawToken == nil || rawToken.(string) == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Authorization token missing or invalid"})
		return
	}

	userIDVal, _ := c.Get("userID")
	var userID int64 = 1
	if id, ok := userIDVal.(int); ok && id > 0 {
		userID = int64(id)
	} else if id64, ok := userIDVal.(int64); ok && id64 > 0 {
		userID = id64
	} else {
		handlerLogger.Warn("Could not get valid UserID from context (expected from middleware), using placeholder 1")
	}
	handlerLogger.Infof("Handling CreateOrder for (placeholder/context) UserID: %d", userID)

	if err := c.ShouldBindJSON(&req); err != nil {
		handlerLogger.Warnf("Failed to bind request: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body: " + err.Error()})
		return
	}

	// Map items
	grpcItems := make([]*orderpb.OrderItem, 0, len(req.Items))
	for _, item := range req.Items {
		grpcItems = append(grpcItems, &orderpb.OrderItem{
			ProductId: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	grpcReq := &orderpb.CreateOrderRequest{
		UserId: userID,
		Items:  grpcItems,
	}

	ctxWithMD := getContextWithAuthToken(c)
	callCtx, cancel := context.WithTimeout(ctxWithMD, 10*time.Second)
	defer cancel()

	grpcRes, err := h.orderClient.CreateOrder(callCtx, grpcReq)
	if err != nil {
		mapGrpcErrorToHttpStatus(c, handlerLogger, err)
		return
	}

	c.JSON(http.StatusCreated, grpcRes)
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	handlerLogger := h.log.WithField("handler", "GetOrder")
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		handlerLogger.Warnf("Invalid order ID parameter: %s", idStr)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid order ID format"})
		return
	}

	grpcReq := &orderpb.GetOrderRequest{Id: id}

	ctxWithMD := getContextWithAuthToken(c)
	callCtx, cancel := context.WithTimeout(ctxWithMD, 5*time.Second)
	defer cancel()

	grpcRes, err := h.orderClient.GetOrder(callCtx, grpcReq)
	if err != nil {
		mapGrpcErrorToHttpStatus(c, handlerLogger, err)
		return
	}

	//Here i will add aspect method before auth response sending

	c.JSON(http.StatusOK, grpcRes)
}

func (h *OrderHandler) ListOrders(c *gin.Context) {
	handlerLogger := h.log.WithField("handler", "ListOrders")

	rawToken, _ := c.Get("rawToken")
	rawToken = rawToken.(string)
	userIDVal, _ := c.Get("userID")
	var userID int64 = 1
	if id, ok := userIDVal.(int); ok && id > 0 {
		userID = int64(id)
	} else if id64, ok := userIDVal.(int64); ok && id64 > 0 {
		userID = id64
	} else {
		handlerLogger.Warn("Could not get valid UserID from context (expected from middleware), using placeholder 1 for ListOrders")

	}
	handlerLogger.Infof("Handling ListOrders for (placeholder/context) UserID: %d", userID)

	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")
	limit, err := strconv.ParseInt(limitStr, 10, 32)
	if err != nil || limit < 0 {
		limit = 10
	}
	offset, err := strconv.ParseInt(offsetStr, 10, 32)
	if err != nil || offset < 0 {
		offset = 0
	}

	grpcReq := &orderpb.ListOrdersRequest{
		UserId: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	ctxWithMD := getContextWithAuthToken(c)
	callCtx, cancel := context.WithTimeout(ctxWithMD, 5*time.Second)
	defer cancel()

	grpcRes, err := h.orderClient.ListOrders(callCtx, grpcReq)
	if err != nil {
		mapGrpcErrorToHttpStatus(c, handlerLogger, err)
		return
	}

	c.JSON(http.StatusOK, grpcRes)
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=pending completed cancelled"`
}

func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	handlerLogger := h.log.WithField("handler", "UpdateOrderStatus")
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		handlerLogger.Warnf("Invalid order ID parameter: %s", idStr)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid order ID format"})
		return
	}

	var req UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerLogger.Warnf("Failed to bind request: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body: " + err.Error()})
		return
	}

	var protoStatus orderpb.OrderStatus
	switch req.Status {
	case "pending":
		protoStatus = orderpb.OrderStatus_PENDING
	case "completed":
		protoStatus = orderpb.OrderStatus_COMPLETED
	case "cancelled":
		protoStatus = orderpb.OrderStatus_CANCELLED
	default:
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid status value"})
		return
	}

	grpcReq := &orderpb.UpdateOrderStatusRequest{
		Id:     id,
		Status: protoStatus,
	}

	ctxWithMD := getContextWithAuthToken(c)
	callCtx, cancel := context.WithTimeout(ctxWithMD, 10*time.Second)
	defer cancel()

	grpcRes, err := h.orderClient.UpdateOrderStatus(callCtx, grpcReq)
	if err != nil {
		mapGrpcErrorToHttpStatus(c, handlerLogger, err)
		return
	}

	c.JSON(http.StatusOK, grpcRes)
}
