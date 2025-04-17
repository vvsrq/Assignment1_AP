package clients

import (
	"context"
	"fmt"
	inventorypb "order_service/proto/inventorypb"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type Product struct {
	ID    int
	Name  string
	Price float64
	Stock int
}

type InventoryClient interface {
	GetProduct(ctx context.Context, productID int) (*Product, error)
	UpdateStock(ctx context.Context, productID int, newStock int) error
}

type inventoryGRPCClient struct {
	client inventorypb.InventoryServiceClient
	log    *logrus.Logger
	conn   *grpc.ClientConn // Keep connection to close it later
}

func NewInventoryGRPCClient(target string, logger *logrus.Logger, timeout time.Duration) (InventoryClient, error) {
	logger.Infof("InventoryClient: Dialing gRPC target: %s", target)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		logger.Errorf("InventoryClient: Failed to dial %s: %v", target, err)
		return nil, fmt.Errorf("failed to connect to inventory service at %s: %w", target, err)
	}
	logger.Infof("InventoryClient: gRPC connection established to %s", target)

	grpcClient := inventorypb.NewInventoryServiceClient(conn)

	return &inventoryGRPCClient{
		client: grpcClient,
		log:    logger,
		conn:   conn,
	}, nil
}

func (c *inventoryGRPCClient) Close() error {
	if c.conn != nil {
		c.log.Info("InventoryClient: Closing gRPC connection")
		return c.conn.Close()
	}
	return nil
}

func (c *inventoryGRPCClient) GetProduct(ctx context.Context, productID int) (*Product, error) {
	c.log.Infof("InventoryClient(gRPC): Requesting product info for ID: %d", productID)
	req := &inventorypb.GetProductRequest{Id: int64(productID)}

	callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	res, err := c.client.GetProduct(callCtx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {

			if st.Code() == codes.NotFound {
				c.log.Warnf("InventoryClient(gRPC): Product with ID %d not found", productID)
				return nil, fmt.Errorf("product with ID %d not found in inventory", productID)
			}
			c.log.Errorf("InventoryClient(gRPC): GetProduct failed for ID %d with code %s: %s", productID, st.Code(), st.Message())
			return nil, fmt.Errorf("inventory service gRPC error (%s): %s", st.Code(), st.Message())
		}

		c.log.Errorf("InventoryClient(gRPC): Failed to execute GetProduct request for ID %d: %v", productID, err)
		return nil, fmt.Errorf("failed to communicate with inventory service: %w", err)
	}

	product := &Product{
		ID:    int(res.GetId()),
		Name:  res.GetName(),
		Price: res.GetPrice(),
		Stock: int(res.GetStock()),
	}

	c.log.Infof("InventoryClient(gRPC): Parsed product data for ID %d: Name='%s', Stock=%d",
		productID, product.Name, product.Stock)

	return product, nil
}

func (c *inventoryGRPCClient) UpdateStock(ctx context.Context, productID int, newStock int) error {
	c.log.Infof("InventoryClient(gRPC): Requesting stock update for ID %d to %d", productID, newStock)
	if newStock < 0 {
		return fmt.Errorf("stock cannot be negative") // Basic validation
	}

	req := &inventorypb.UpdateProductRequest{
		Product: &inventorypb.Product{
			Id:    int64(productID),
			Stock: int32(newStock),
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"stock"},
		},
	}

	callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := c.client.UpdateProduct(callCtx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.NotFound {
				c.log.Warnf("InventoryClient(gRPC): Product with ID %d not found for stock update", productID)
				return fmt.Errorf("product with ID %d not found in inventory for update", productID)
			}
			if st.Code() == codes.InvalidArgument {
				c.log.Warnf("InventoryClient(gRPC): Invalid request updating stock for ID %d: %s", productID, st.Message())
				return fmt.Errorf("invalid stock update request for product %d: %s", productID, st.Message())
			}
			c.log.Errorf("InventoryClient(gRPC): UpdateProduct failed for ID %d with code %s: %s", productID, st.Code(), st.Message())
			return fmt.Errorf("inventory service gRPC error (%s): %s", st.Code(), st.Message())
		}
		c.log.Errorf("InventoryClient(gRPC): Failed to execute UpdateStock request for ID %d: %v", productID, err)
		return fmt.Errorf("failed to communicate with inventory service for stock update: %w", err)
	}

	c.log.Infof("InventoryClient(gRPC): Successfully updated stock for product ID %d to %d", productID, newStock)
	return nil
}
