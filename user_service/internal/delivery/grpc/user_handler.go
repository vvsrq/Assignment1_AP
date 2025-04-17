package grpc

import (
	"context"
	"strings"
	"user_service/internal/domain"
	userpb "user_service/proto"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserHandler struct {
	userpb.UnimplementedUserServiceServer
	useCase domain.UserUseCase
	log     *logrus.Logger
}

func NewUserHandler(uc domain.UserUseCase, logger *logrus.Logger) *UserHandler {
	return &UserHandler{
		useCase: uc,
		log:     logger,
	}
}

func (h *UserHandler) RegisterUser(ctx context.Context, req *userpb.RegisterUserRequest) (*userpb.User, error) {
	h.log.Infof("gRPC Handler: Received RegisterUser request for email: %s", req.GetEmail())

	if req.GetName() == "" || req.GetEmail() == "" || req.GetPassword() == "" {
		h.log.Warn("gRPC Handler: RegisterUser validation failed - missing fields")
		return nil, status.Error(codes.InvalidArgument, "Name, email, and password are required")
	}

	createdUser, err := h.useCase.RegisterUser(req.GetName(), req.GetEmail(), req.GetPassword())
	if err != nil {
		h.log.Errorf("gRPC Handler: RegisterUser use case failed: %v", err)

		if strings.Contains(err.Error(), "already exists") {
			return nil, status.Errorf(codes.AlreadyExists, "User registration failed: %v", err)
		}
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "must contain") || strings.Contains(err.Error(), "characters long") {

			return nil, status.Errorf(codes.InvalidArgument, "User registration failed: %v", err)
		}

		return nil, status.Errorf(codes.Internal, "User registration failed: %v", err)
	}

	response := &userpb.User{
		Id:    createdUser.ID,
		Name:  createdUser.Name,
		Email: createdUser.Email,
	}

	h.log.Infof("gRPC Handler: RegisterUser successful for ID: %d", response.Id)
	return response, nil
}

func (h *UserHandler) AuthenticateUser(ctx context.Context, req *userpb.AuthenticateUserRequest) (*userpb.AuthenticateUserResponse, error) {
	h.log.Infof("gRPC Handler: Received AuthenticateUser request for email: %s", req.GetEmail())

	if req.GetEmail() == "" || req.GetPassword() == "" {
		h.log.Warn("gRPC Handler: AuthenticateUser validation failed - missing fields")
		return nil, status.Error(codes.InvalidArgument, "Email and password are required")
	}

	authResult, err := h.useCase.AuthenticateUser(req.GetEmail(), req.GetPassword())
	if err != nil {

		h.log.Errorf("gRPC Handler: AuthenticateUser use case internal error: %v", err)
		return nil, status.Errorf(codes.Internal, "Authentication failed due to an internal error: %v", err)
	}

	if !authResult.Authenticated {
		h.log.Warnf("gRPC Handler: Authentication failed for email %s: %s", req.GetEmail(), authResult.ErrorMessage)

		return &userpb.AuthenticateUserResponse{
			Authenticated: false,
			ErrorMessage:  authResult.ErrorMessage,
		}, nil

	}

	response := &userpb.AuthenticateUserResponse{
		Authenticated: true,
		Token:         authResult.Token,
		UserId:        authResult.UserID,
		ErrorMessage:  "",
	}

	h.log.Infof("gRPC Handler: AuthenticateUser successful for User ID: %d", response.UserId)
	return response, nil
}

func (h *UserHandler) GetUserProfile(ctx context.Context, req *userpb.GetUserProfileRequest) (*userpb.UserProfile, error) {
	userID := req.GetUserId()
	h.log.Infof("gRPC Handler: Received GetUserProfile request for User ID: %d", userID)

	if userID <= 0 {
		h.log.Warn("gRPC Handler: GetUserProfile validation failed - invalid user ID")
		return nil, status.Error(codes.InvalidArgument, "Valid User ID is required")
	}

	profile, err := h.useCase.GetUserProfile(userID)
	if err != nil {
		h.log.Warnf("gRPC Handler: GetUserProfile use case failed for User ID %d: %v", userID, err)

		if strings.Contains(err.Error(), "not found") {
			return nil, status.Errorf(codes.NotFound, "User profile not found: %v", err)
		}

		return nil, status.Errorf(codes.Internal, "Failed to retrieve user profile: %v", err)
	}

	response := &userpb.UserProfile{
		Id:    profile.ID,
		Name:  profile.Name,
		Email: profile.Email,
	}

	h.log.Infof("gRPC Handler: GetUserProfile successful for User ID: %d", response.Id)
	return response, nil
}
