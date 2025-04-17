package handlers

import (
	"api_gateway/internal/clients"
	userpb "api_gateway/proto/userpb"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type AuthHandler struct {
	userClient clients.UserServiceClient
	log        *logrus.Logger
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(uc clients.UserServiceClient, logger *logrus.Logger) *AuthHandler {
	return &AuthHandler{
		userClient: uc,
		log:        logger,
	}
}

// LoginRequest defines the expected JSON body for login requests
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse defines the JSON response for successful login
type LoginResponse struct {
	Token string `json:"token"`
}

// Login handles the POST /login request
func (h *AuthHandler) Login(c *gin.Context) {
	handlerLogger := h.log.WithField("handler", "Login")
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		handlerLogger.Warnf("Failed to bind login request: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body: " + err.Error()})
		return
	}
	handlerLogger.Infof("Processing login request for email: %s", req.Email)

	grpcReq := &userpb.AuthenticateUserRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	grpcRes, err := h.userClient.AuthenticateUser(ctx, grpcReq)

	if err != nil {
		mapGrpcErrorToHttpStatus(c, h.log, err)
		return
	}

	if !grpcRes.GetAuthenticated() {
		handlerLogger.Warnf("Authentication failed for email %s: %s", req.Email, grpcRes.GetErrorMessage())
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: grpcRes.GetErrorMessage()})
		return
	}

	handlerLogger.Infof("Authentication successful for UserID: %d", grpcRes.GetUserId())
	c.JSON(http.StatusOK, LoginResponse{Token: grpcRes.GetToken()})
}
