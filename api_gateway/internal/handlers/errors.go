package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func mapGrpcErrorToHttpStatus(c *gin.Context, logger logrus.FieldLogger, grpcError error) {
	st, ok := status.FromError(grpcError)
	if !ok {
		logger.Errorf("Handler Error: Non-gRPC error encountered: %v", grpcError)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	var httpStatus int
	var clientMessage string

	switch st.Code() {
	case codes.OK:
		httpStatus = http.StatusOK
		clientMessage = "Success (unexpected error state)"
	case codes.InvalidArgument:
		httpStatus = http.StatusBadRequest
		clientMessage = st.Message()
	case codes.NotFound:
		httpStatus = http.StatusNotFound
		clientMessage = st.Message()
	case codes.AlreadyExists:
		httpStatus = http.StatusConflict
		clientMessage = st.Message()
	case codes.PermissionDenied:
		httpStatus = http.StatusForbidden
		clientMessage = st.Message()
	case codes.Unauthenticated:
		httpStatus = http.StatusUnauthorized
		clientMessage = st.Message()
	case codes.ResourceExhausted:
		httpStatus = http.StatusTooManyRequests
		clientMessage = st.Message()
	case codes.FailedPrecondition:
		httpStatus = http.StatusBadRequest
		clientMessage = st.Message()
	case codes.Aborted:
		httpStatus = http.StatusConflict
		clientMessage = st.Message()
	case codes.OutOfRange:
		httpStatus = http.StatusBadRequest // 400
		clientMessage = st.Message()
	case codes.Unimplemented:
		httpStatus = http.StatusNotImplemented // 501
		clientMessage = "Feature not implemented"
	case codes.Internal:
		httpStatus = http.StatusInternalServerError // 500
		clientMessage = "Internal server error"
	case codes.Unavailable:
		httpStatus = http.StatusServiceUnavailable // 503
		clientMessage = "Service temporarily unavailable"
	case codes.DataLoss:
		httpStatus = http.StatusInternalServerError // 500
		clientMessage = "Internal server error (data loss)"
	case codes.DeadlineExceeded:
		httpStatus = http.StatusGatewayTimeout // 504
		clientMessage = "Request timed out"
	default:
		httpStatus = http.StatusInternalServerError // 500
		clientMessage = "An unexpected error occurred"
	}

	logger.Warnf("Handler Error: Mapped gRPC error (Code: %s, Message: '%s') to HTTP Status %d", st.Code(), st.Message(), httpStatus)

	c.JSON(httpStatus, ErrorResponse{Error: clientMessage})
}
