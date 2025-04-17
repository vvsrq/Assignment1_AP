package clients

import (
	inventorypb "api_gateway/proto/inventorypb"
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

type InventoryServiceClient interface {
	CreateCategory(ctx context.Context, req *inventorypb.CreateCategoryRequest) (*inventorypb.Category, error)
	GetCategory(ctx context.Context, req *inventorypb.GetCategoryRequest) (*inventorypb.Category, error)
	UpdateCategory(ctx context.Context, req *inventorypb.UpdateCategoryRequest) (*inventorypb.Category, error)
	DeleteCategory(ctx context.Context, req *inventorypb.DeleteCategoryRequest) (*emptypb.Empty, error)
	ListCategories(ctx context.Context, req *inventorypb.ListCategoriesRequest) (*inventorypb.ListCategoriesResponse, error)

	CreateProduct(ctx context.Context, req *inventorypb.CreateProductRequest) (*inventorypb.Product, error)
	GetProduct(ctx context.Context, req *inventorypb.GetProductRequest) (*inventorypb.Product, error)
	UpdateProduct(ctx context.Context, req *inventorypb.UpdateProductRequest) (*inventorypb.Product, error)
	DeleteProduct(ctx context.Context, req *inventorypb.DeleteProductRequest) (*emptypb.Empty, error)
	ListProducts(ctx context.Context, req *inventorypb.ListProductsRequest) (*inventorypb.ListProductsResponse, error)

	Close() error
}

type inventoryGRPCClient struct {
	client inventorypb.InventoryServiceClient
	conn   *grpc.ClientConn
	log    *logrus.Logger
}

func NewInventoryServiceClient(target string, logger *logrus.Logger, timeout time.Duration) (InventoryServiceClient, error) {
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
		conn:   conn,
		log:    logger,
	}, nil
}

func (c *inventoryGRPCClient) Close() error {
	if c.conn != nil {
		c.log.Info("InventoryClient: Closing gRPC connection")
		return c.conn.Close()
	}
	return nil
}

func (c *inventoryGRPCClient) CreateCategory(ctx context.Context, req *inventorypb.CreateCategoryRequest) (*inventorypb.Category, error) {
	c.log.Debugf("InventoryClient(gRPC): Calling CreateCategory: Name=%s", req.GetName())
	return c.client.CreateCategory(ctx, req)
}

func (c *inventoryGRPCClient) GetCategory(ctx context.Context, req *inventorypb.GetCategoryRequest) (*inventorypb.Category, error) {
	c.log.Debugf("InventoryClient(gRPC): Calling GetCategory: ID=%d", req.GetId())
	return c.client.GetCategory(ctx, req)
}

func (c *inventoryGRPCClient) UpdateCategory(ctx context.Context, req *inventorypb.UpdateCategoryRequest) (*inventorypb.Category, error) {
	catID := int64(0)
	if req.GetCategory() != nil {
		catID = req.GetCategory().GetId()
	}
	c.log.Debugf("InventoryClient(gRPC): Calling UpdateCategory: ID=%d", catID)
	return c.client.UpdateCategory(ctx, req)
}

func (c *inventoryGRPCClient) DeleteCategory(ctx context.Context, req *inventorypb.DeleteCategoryRequest) (*emptypb.Empty, error) {
	c.log.Debugf("InventoryClient(gRPC): Calling DeleteCategory: ID=%d", req.GetId())
	return c.client.DeleteCategory(ctx, req)
}

func (c *inventoryGRPCClient) ListCategories(ctx context.Context, req *inventorypb.ListCategoriesRequest) (*inventorypb.ListCategoriesResponse, error) {
	c.log.Debugf("InventoryClient(gRPC): Calling ListCategories")
	return c.client.ListCategories(ctx, req)
}

func (c *inventoryGRPCClient) CreateProduct(ctx context.Context, req *inventorypb.CreateProductRequest) (*inventorypb.Product, error) {
	c.log.Debugf("InventoryClient(gRPC): Calling CreateProduct: Name=%s", req.GetName())
	return c.client.CreateProduct(ctx, req)
}

func (c *inventoryGRPCClient) GetProduct(ctx context.Context, req *inventorypb.GetProductRequest) (*inventorypb.Product, error) {
	c.log.Debugf("InventoryClient(gRPC): Calling GetProduct: ID=%d", req.GetId())
	return c.client.GetProduct(ctx, req)
}

func (c *inventoryGRPCClient) UpdateProduct(ctx context.Context, req *inventorypb.UpdateProductRequest) (*inventorypb.Product, error) {
	prodID := int64(0)
	if req.GetProduct() != nil {
		prodID = req.GetProduct().GetId()
	}
	c.log.Debugf("InventoryClient(gRPC): Calling UpdateProduct: ID=%d", prodID)
	return c.client.UpdateProduct(ctx, req)
}

func (c *inventoryGRPCClient) DeleteProduct(ctx context.Context, req *inventorypb.DeleteProductRequest) (*emptypb.Empty, error) {
	c.log.Debugf("InventoryClient(gRPC): Calling DeleteProduct: ID=%d", req.GetId())
	return c.client.DeleteProduct(ctx, req)
}

func (c *inventoryGRPCClient) ListProducts(ctx context.Context, req *inventorypb.ListProductsRequest) (*inventorypb.ListProductsResponse, error) {
	c.log.Debugf("InventoryClient(gRPC): Calling ListProducts: Limit=%d, Offset=%d", req.GetLimit(), req.GetOffset())
	return c.client.ListProducts(ctx, req)
}
