package grpc

import (
	"context"
	"order_service/internal/domain"

	orderpb "order_service/proto"
	"strings"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderHandler struct {
	orderpb.UnimplementedOrderServiceServer
	useCase domain.OrderUseCase
	log     *logrus.Logger
}

func NewOrderHandler(uc domain.OrderUseCase, logger *logrus.Logger) *OrderHandler {
	return &OrderHandler{
		useCase: uc,
		log:     logger,
	}
}

func mapProtoStatusToDomain(protoStatus orderpb.OrderStatus) domain.OrderStatus {
	switch protoStatus {
	case orderpb.OrderStatus_PENDING:
		return domain.StatusPending
	case orderpb.OrderStatus_COMPLETED:
		return domain.StatusCompleted
	case orderpb.OrderStatus_CANCELLED:
		return domain.StatusCancelled
	default:
		return ""
	}
}

func mapDomainStatusToProto(domainStatus domain.OrderStatus) orderpb.OrderStatus {
	switch domainStatus {
	case domain.StatusPending:
		return orderpb.OrderStatus_PENDING
	case domain.StatusCompleted:
		return orderpb.OrderStatus_COMPLETED
	case domain.StatusCancelled:
		return orderpb.OrderStatus_CANCELLED
	default:
		return orderpb.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

func mapProtoItemsToDomain(protoItems []*orderpb.OrderItem) []domain.OrderItem {
	domainItems := make([]domain.OrderItem, 0, len(protoItems))
	for _, item := range protoItems {
		if item != nil {
			domainItems = append(domainItems, domain.OrderItem{
				ProductID: int(item.GetProductId()),
				Quantity:  int(item.GetQuantity()),
				Price:     item.GetPrice(),
			})
		}
	}
	return domainItems
}

func mapDomainItemsToProto(domainItems []domain.OrderItem) []*orderpb.OrderItem {
	protoItems := make([]*orderpb.OrderItem, 0, len(domainItems))
	for _, item := range domainItems {
		protoItems = append(protoItems, &orderpb.OrderItem{
			ProductId: int64(item.ProductID),
			Quantity:  int32(item.Quantity),
			Price:     item.Price,
		})
	}
	return protoItems
}

func mapDomainOrderToProto(order *domain.Order) *orderpb.Order {
	if order == nil {
		return nil
	}
	return &orderpb.Order{
		Id:        int64(order.ID),
		UserId:    int64(order.UserID),
		Items:     mapDomainItemsToProto(order.Items),
		Status:    mapDomainStatusToProto(order.Status),
		CreatedAt: timestamppb.New(order.CreatedAt),
		UpdatedAt: timestamppb.New(order.UpdatedAt),
	}
}

func (h *OrderHandler) CreateOrder(ctx context.Context, req *orderpb.CreateOrderRequest) (*orderpb.Order, error) {
	userID := req.GetUserId()
	h.log.Infof("gRPC Handler: Received CreateOrder request for UserID: %d with %d items", userID, len(req.GetItems()))

	if userID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "Invalid User ID provided")
	}
	if len(req.GetItems()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Order must contain at least one item")
	}

	domainOrder := &domain.Order{
		UserID: int(userID),
		Items:  mapProtoItemsToDomain(req.GetItems()),
	}

	createdOrder, err := h.useCase.CreateOrder(ctx, domainOrder)
	if err != nil {
		h.log.Errorf("gRPC Handler: CreateOrder use case error for UserID %d: %v", userID, err)

		return nil, mapOrderDomainErrorToGrpcStatus(err)
	}

	h.log.Infof("gRPC Handler: Order created successfully: OrderID=%d for UserID=%d", createdOrder.ID, createdOrder.UserID)
	return mapDomainOrderToProto(createdOrder), nil
}

func (h *OrderHandler) GetOrder(ctx context.Context, req *orderpb.GetOrderRequest) (*orderpb.Order, error) {
	orderID := int(req.GetId())
	h.log.Infof("gRPC Handler: Received GetOrder request for OrderID: %d", orderID)

	// TODO (Future): Extract UserID from context metadata for authorization check

	if orderID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "Invalid Order ID")
	}
	order, err := h.useCase.GetOrderByID(orderID)
	if err != nil {
		h.log.Warnf("gRPC Handler: GetOrderByID use case error for OrderID %d: %v", orderID, err)
		return nil, mapOrderDomainErrorToGrpcStatus(err)
	}

	// TODO (Future): Perform authorization check here

	h.log.Infof("gRPC Handler: Order retrieved successfully: OrderID=%d", order.ID)
	return mapDomainOrderToProto(order), nil
}

func (h *OrderHandler) UpdateOrderStatus(ctx context.Context, req *orderpb.UpdateOrderStatusRequest) (*orderpb.Order, error) {
	orderID := int(req.GetId())
	newStatus := mapProtoStatusToDomain(req.GetStatus())
	h.log.Infof("gRPC Handler: Received UpdateOrderStatus request for OrderID: %d to Status: %s", orderID, newStatus)

	// TODO (Future): Extract UserID from context metadata for authorization check

	if orderID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "Invalid Order ID")
	}
	if !domain.IsValidStatus(newStatus) {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid status value provided: %s", newStatus)
	}

	// TODO (Future): Get current order first to check ownership *before* calling update use case

	updatedOrder, err := h.useCase.UpdateOrderStatus(ctx, orderID, newStatus)
	if err != nil {
		h.log.Errorf("gRPC Handler: UpdateOrderStatus use case error for OrderID %d: %v", orderID, err)
		return nil, mapOrderDomainErrorToGrpcStatus(err)
	}

	h.log.Infof("gRPC Handler: Order status updated successfully: OrderID=%d to Status=%s", updatedOrder.ID, updatedOrder.Status)
	return mapDomainOrderToProto(updatedOrder), nil
}

func (h *OrderHandler) ListOrders(ctx context.Context, req *orderpb.ListOrdersRequest) (*orderpb.ListOrdersResponse, error) {
	userID := int(req.GetUserId())
	limit := int(req.GetLimit())
	offset := int(req.GetOffset())
	h.log.Infof("gRPC Handler: Received ListOrders request for UserID: %d, Limit: %d, Offset: %d", userID, limit, offset)

	// TODO (Future): Compare userID from request with UserID from metadata for authorization

	if userID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "Invalid User ID")
	}

	orders, err := h.useCase.ListOrdersByUserID(userID, limit, offset)
	if err != nil {
		h.log.Errorf("gRPC Handler: ListOrdersByUserID use case error for UserID %d: %v", userID, err)
		return nil, mapOrderDomainErrorToGrpcStatus(err)
	}

	resp := &orderpb.ListOrdersResponse{
		Orders: make([]*orderpb.Order, 0, len(orders)),
	}
	for i := range orders {
		resp.Orders = append(resp.Orders, mapDomainOrderToProto(&orders[i]))
	}

	h.log.Infof("gRPC Handler: Listed %d orders successfully for UserID %d", len(resp.Orders), userID)
	return resp, nil
}

func mapOrderDomainErrorToGrpcStatus(err error) error {
	if err == nil {
		return nil
	}
	errMsg := strings.ToLower(err.Error())

	if strings.Contains(errMsg, "insufficient stock") {
		return status.Error(codes.FailedPrecondition, err.Error()) // Or ResourceExhausted?
	}
	if strings.Contains(errMsg, "inventory check failed") && strings.Contains(errMsg, "not found") {

		return status.Error(codes.NotFound, err.Error())
	}
	if strings.Contains(errMsg, "cannot cancel a completed order") || strings.Contains(errMsg, "cannot change status of a cancelled order") {
		return status.Error(codes.FailedPrecondition, err.Error())
	}

	switch {
	case strings.Contains(errMsg, "not found"):
		return status.Error(codes.NotFound, err.Error())
	case strings.Contains(errMsg, "invalid") || strings.Contains(errMsg, "must contain") || strings.Contains(errMsg, "cannot be empty"):
		return status.Error(codes.InvalidArgument, err.Error())

	default:

		return status.Errorf(codes.Internal, "Internal server error: %v", err)
	}
}
