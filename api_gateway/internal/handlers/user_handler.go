package handlers

import (
	"api_gateway/internal/clients"
	userpb "api_gateway/proto/userpb"
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type UserHandler struct {
	userClient clients.UserServiceClient
	log        *logrus.Logger
}

func NewUserHandler(uc clients.UserServiceClient, logger *logrus.Logger) *UserHandler {
	return &UserHandler{
		userClient: uc,
		log:        logger,
	}
}

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type RegisterResponse struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (h *UserHandler) Register(c *gin.Context) {
	handlerLogger := h.log.WithField("handler", "Register")
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		handlerLogger.Warnf("Failed to bind register request: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body: " + err.Error()})
		return
	}
	handlerLogger.Infof("Processing registration request for email: %s", req.Email)

	grpcReq := &userpb.RegisterUserRequest{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	grpcRes, err := h.userClient.RegisterUser(ctx, grpcReq)
	if err != nil {
		mapGrpcErrorToHttpStatus(c, h.log, err)
		return
	}

	handlerLogger.Infof("Registration successful for UserID: %d", grpcRes.GetId())
	resp := RegisterResponse{
		ID:    grpcRes.GetId(),
		Name:  grpcRes.GetName(),
		Email: grpcRes.GetEmail(),
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	handlerLogger := h.log.WithField("handler", "GetProfile")

	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil || userID <= 0 {
		handlerLogger.Warnf("Invalid User ID in path parameter: %s", userIDStr)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid user ID format in URL"})
		return
	}
	handlerLogger.Infof("Requesting profile for UserID from URL: %d", userID)

	grpcReq := &userpb.GetUserProfileRequest{
		UserId: userID,
	}

	ctxWithMD := getContextWithAuthToken(c)
	callCtx, cancel := context.WithTimeout(ctxWithMD, 5*time.Second)
	defer cancel()

	grpcRes, err := h.userClient.GetUserProfile(callCtx, grpcReq)
	if err != nil {

		mapGrpcErrorToHttpStatus(c, handlerLogger, err)
		return
	}

	handlerLogger.Infof("Profile retrieved successfully for UserID: %d", grpcRes.GetId())
	c.JSON(http.StatusOK, grpcRes)
}
