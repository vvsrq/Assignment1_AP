package clients

import (
	userpb "api_gateway/proto/userpb"
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type UserServiceClient interface {
	AuthenticateUser(ctx context.Context, req *userpb.AuthenticateUserRequest) (*userpb.AuthenticateUserResponse, error)
	RegisterUser(ctx context.Context, req *userpb.RegisterUserRequest) (*userpb.User, error)
	GetUserProfile(ctx context.Context, req *userpb.GetUserProfileRequest) (*userpb.UserProfile, error)
	Close() error
}

type userServiceGRPCClient struct {
	client userpb.UserServiceClient
	conn   *grpc.ClientConn
	log    *logrus.Logger
}

func NewUserServiceClient(target string, logger *logrus.Logger, timeout time.Duration) (UserServiceClient, error) {
	logger.Infof("UserClient: Dialing gRPC target: %s", target)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		logger.Errorf("UserClient: Failed to dial %s: %v", target, err)
		return nil, fmt.Errorf("failed to connect to user service at %s: %w", target, err)
	}
	logger.Infof("UserClient: gRPC connection established to %s", target)

	grpcClient := userpb.NewUserServiceClient(conn)

	return &userServiceGRPCClient{
		client: grpcClient,
		conn:   conn,
		log:    logger,
	}, nil
}

func (c *userServiceGRPCClient) Close() error {
	if c.conn != nil {
		c.log.Info("UserClient: Closing gRPC connection")
		return c.conn.Close()
	}
	return nil
}

func (c *userServiceGRPCClient) AuthenticateUser(ctx context.Context, req *userpb.AuthenticateUserRequest) (*userpb.AuthenticateUserResponse, error) {
	c.log.Debugf("UserClient(gRPC): Calling AuthenticateUser for email: %s", req.GetEmail())
	return c.client.AuthenticateUser(ctx, req)
}

func (c *userServiceGRPCClient) RegisterUser(ctx context.Context, req *userpb.RegisterUserRequest) (*userpb.User, error) {
	c.log.Debugf("UserClient(gRPC): Calling RegisterUser for email: %s", req.GetEmail())
	return c.client.RegisterUser(ctx, req)
}

func (c *userServiceGRPCClient) GetUserProfile(ctx context.Context, req *userpb.GetUserProfileRequest) (*userpb.UserProfile, error) {
	c.log.Debugf("UserClient(gRPC): Calling GetUserProfile for UserID: %d", req.GetUserId())
	return c.client.GetUserProfile(ctx, req)
}
