package clients

import (
	orderpb "api_gateway/proto/orderpb"
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type OrderServiceClient interface {
	CreateOrder(ctx context.Context, req *orderpb.CreateOrderRequest) (*orderpb.Order, error)
	GetOrder(ctx context.Context, req *orderpb.GetOrderRequest) (*orderpb.Order, error)
	UpdateOrderStatus(ctx context.Context, req *orderpb.UpdateOrderStatusRequest) (*orderpb.Order, error)
	ListOrders(ctx context.Context, req *orderpb.ListOrdersRequest) (*orderpb.ListOrdersResponse, error)
	Close() error
}

type orderGRPCClient struct {
	client orderpb.OrderServiceClient // Сгенерированный клиент Order
	conn   *grpc.ClientConn
	log    *logrus.Logger
}

func NewOrderServiceClient(target string, logger *logrus.Logger, timeout time.Duration) (OrderServiceClient, error) {
	logger.Infof("OrderClient: Dialing gRPC target: %s", target)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		logger.Errorf("OrderClient: Failed to dial %s: %v", target, err)
		return nil, fmt.Errorf("failed to connect to order service at %s: %w", target, err)
	}
	logger.Infof("OrderClient: gRPC connection established to %s", target)

	grpcClient := orderpb.NewOrderServiceClient(conn)

	return &orderGRPCClient{
		client: grpcClient,
		conn:   conn,
		log:    logger,
	}, nil
}

func (c *orderGRPCClient) Close() error {
	if c.conn != nil {
		c.log.Info("OrderClient: Closing gRPC connection")
		return c.conn.Close()
	}
	return nil
}

func (c *orderGRPCClient) CreateOrder(ctx context.Context, req *orderpb.CreateOrderRequest) (*orderpb.Order, error) {
	c.log.Debugf("OrderClient(gRPC): Calling CreateOrder for UserID: %d", req.GetUserId())
	return c.client.CreateOrder(ctx, req)
}

func (c *orderGRPCClient) GetOrder(ctx context.Context, req *orderpb.GetOrderRequest) (*orderpb.Order, error) {
	c.log.Debugf("OrderClient(gRPC): Calling GetOrder for OrderID: %d", req.GetId())
	return c.client.GetOrder(ctx, req)
}

func (c *orderGRPCClient) UpdateOrderStatus(ctx context.Context, req *orderpb.UpdateOrderStatusRequest) (*orderpb.Order, error) {
	c.log.Debugf("OrderClient(gRPC): Calling UpdateOrderStatus for OrderID: %d to Status: %s", req.GetId(), req.GetStatus())
	return c.client.UpdateOrderStatus(ctx, req)
}

func (c *orderGRPCClient) ListOrders(ctx context.Context, req *orderpb.ListOrdersRequest) (*orderpb.ListOrdersResponse, error) {
	c.log.Debugf("OrderClient(gRPC): Calling ListOrders for UserID: %d", req.GetUserId())
	return c.client.ListOrders(ctx, req)
}
