package delivery

import (
	"fmt"
	"net/http"
	"order_service/internal/domain"
	"order_service/internal/usecase"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type OrderHandler struct {
	useCase usecase.OrderUseCase
	log     *logrus.Logger
}

func NewOrderHandler(uc usecase.OrderUseCase, logger *logrus.Logger) *OrderHandler {
	return &OrderHandler{
		useCase: uc,
		log:     logger,
	}
}

func (h *OrderHandler) RegisterRoutes(router gin.IRouter) {
	orders := router.Group("/orders")
	{
		orders.POST("", h.CreateOrder)
		orders.GET("/:id", h.GetOrderByID)
		orders.PATCH("/:id", h.UpdateOrder)
		orders.GET("", h.ListOrders)
	}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		h.log.Error("X-User-ID header is missing")
		ErrorResponse(c, http.StatusUnauthorized, "User identification missing")
		return
	}
	userID, err := strconv.Atoi(userIDStr)
	if err != nil || userID <= 0 {
		h.log.Errorf("Invalid X-User-ID header value: %s", userIDStr)
		ErrorResponse(c, http.StatusInternalServerError, "Invalid user identification data")
		return
	}
	h.log.Infof("Processing create order request for User ID: %d", userID)

	var requestBody struct {
		Items []domain.OrderItem `json:"items"`
	}
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		h.log.Errorf("Failed to bind JSON for create order (User: %d): %v", userID, err)
		ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	if len(requestBody.Items) == 0 {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request body: items cannot be empty")
		return
	}

	order := domain.Order{
		UserID: userID,
		Items:  requestBody.Items,
	}
	createdOrder, err := h.useCase.CreateOrder(&order)
	if err != nil {
		statusCode := mapErrorToStatus(err)
		h.log.Errorf("Failed to create order for user %d: %v", userID, err)
		ErrorResponse(c, statusCode, "Failed to create order: "+err.Error())
		return
	}

	h.log.Infof("Order %d created successfully for user %d", createdOrder.ID, createdOrder.UserID)
	SuccessResponse(c, http.StatusCreated, "Order created successfully", createdOrder)
}

func (h *OrderHandler) GetOrderByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		h.log.Warnf("Invalid order ID parameter: %s", idStr)
		ErrorResponse(c, http.StatusBadRequest, "Invalid order ID format")
		return
	}

	requestorUserIDStr := c.GetHeader("X-User-ID")
	if requestorUserIDStr == "" {
		h.log.Error("X-User-ID header is missing for GetOrderByID")
		ErrorResponse(c, http.StatusUnauthorized, "User identification missing")
		return
	}
	requestorUserID, err := strconv.Atoi(requestorUserIDStr)
	if err != nil || requestorUserID <= 0 {
		h.log.Errorf("Invalid X-User-ID header value for GetOrderByID: %s", requestorUserIDStr)
		ErrorResponse(c, http.StatusInternalServerError, "Invalid user identification data")
		return
	}
	h.log.Infof("User %d requesting order details for Order ID: %d", requestorUserID, id)

	order, err := h.useCase.GetOrderByID(id)
	if err != nil {
		statusCode := mapErrorToStatus(err)
		h.log.Warnf("Failed to get order by ID %d (requested by user %d): %v", id, requestorUserID, err)
		ErrorResponse(c, statusCode, "Failed to retrieve order: "+err.Error())
		return
	}

	if order.UserID != requestorUserID {
		h.log.Warnf("Authorization failed: User %d attempted to access order %d owned by user %d", requestorUserID, id, order.UserID)
		ErrorResponse(c, http.StatusForbidden, "You are not authorized to view this order")
		return
	}

	h.log.Infof("Order %d retrieved successfully for user %d", id, requestorUserID)
	SuccessResponse(c, http.StatusOK, "Order retrieved successfully", order)
}

func (h *OrderHandler) UpdateOrder(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		h.log.Warnf("Invalid order ID parameter for update: %s", idStr)
		ErrorResponse(c, http.StatusBadRequest, "Invalid order ID format")
		return
	}

	var updateRequest struct {
		Status *domain.OrderStatus `json:"status"`
	}
	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		h.log.Warnf("Failed to bind JSON for update order %d: %v", id, err)
		ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	if updateRequest.Status == nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request body: 'status' field is required")
		return
	}
	if !domain.IsValidStatus(*updateRequest.Status) {
		ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("Invalid request body: invalid status value '%s'", *updateRequest.Status))
		return
	}

	requestorUserIDStr := c.GetHeader("X-User-ID")
	if requestorUserIDStr == "" {
		h.log.Error("X-User-ID header is missing for UpdateOrder")
		ErrorResponse(c, http.StatusUnauthorized, "User identification missing")
		return
	}
	requestorUserID, err := strconv.Atoi(requestorUserIDStr)
	if err != nil || requestorUserID <= 0 {
		h.log.Errorf("Invalid X-User-ID header value for UpdateOrder: %s", requestorUserIDStr)
		ErrorResponse(c, http.StatusInternalServerError, "Invalid user identification data")
		return
	}
	h.log.Infof("User %d attempting to update status for order ID %d to '%s'", requestorUserID, id, *updateRequest.Status)

	currentOrder, err := h.useCase.GetOrderByID(id)
	if err != nil {
		statusCode := mapErrorToStatus(err)
		h.log.Warnf("Failed to get order %d for update check (requested by user %d): %v", id, requestorUserID, err)
		ErrorResponse(c, statusCode, "Failed to retrieve order for update: "+err.Error())
		return
	}

	if currentOrder.UserID != requestorUserID {
		h.log.Warnf("Authorization failed: User %d attempted to update order %d owned by user %d", requestorUserID, id, currentOrder.UserID)
		ErrorResponse(c, http.StatusForbidden, "You are not authorized to update this order")
		return
	}

	updatedOrder, err := h.useCase.UpdateOrderStatus(id, *updateRequest.Status)
	if err != nil {
		statusCode := mapErrorToStatus(err)
		h.log.Errorf("Failed to update status for order ID %d (requested by user %d): %v", id, requestorUserID, err)
		ErrorResponse(c, statusCode, "Failed to update order status: "+err.Error())
		return
	}

	h.log.Infof("Order status updated successfully for ID %d by user %d", updatedOrder.ID, requestorUserID)
	SuccessResponse(c, http.StatusOK, "Order status updated successfully", updatedOrder)
}

func (h *OrderHandler) ListOrders(c *gin.Context) {
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		h.log.Error("X-User-ID header is missing for ListOrders")
		ErrorResponse(c, http.StatusUnauthorized, "User identification missing")
		return
	}
	userID, err := strconv.Atoi(userIDStr)
	if err != nil || userID <= 0 {
		h.log.Errorf("Invalid X-User-ID header value for ListOrders: %s", userIDStr)
		ErrorResponse(c, http.StatusInternalServerError, "Invalid user identification data")
		return
	}
	h.log.Infof("Processing list orders request for User ID: %d", userID)

	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 0 {
		h.log.Warnf("Invalid limit parameter '%s' for user %d, using default 10", limitStr, userID)
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		h.log.Warnf("Invalid offset parameter '%s' for user %d, using default 0", offsetStr, userID)
		offset = 0
	}

	h.log.Infof("Attempting to list orders for user %d with limit %d, offset %d", userID, limit, offset)

	orders, err := h.useCase.ListOrdersByUserID(userID, limit, offset)
	if err != nil {
		statusCode := http.StatusInternalServerError
		h.log.Errorf("Failed to list orders for user %d: %v", userID, err)
		ErrorResponse(c, statusCode, "Failed to retrieve orders")
		return
	}

	h.log.Infof("Retrieved %d orders for user %d", len(orders), userID)
	if len(orders) == 0 {
		SuccessResponse(c, http.StatusOK, "No orders found for this user", []domain.Order{})
		return
	}
	SuccessResponse(c, http.StatusOK, "Orders retrieved successfully", orders)
}
