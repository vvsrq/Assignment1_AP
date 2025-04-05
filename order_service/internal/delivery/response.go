package delivery

import (
	"net/http"

	"strings"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Status  string      `json:"Status"`
	Message string      `json:"Message"`
	Data    interface{} `json:"Data,omitempty"`
}

func SuccessResponse(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, Response{
		Status:  "Success",
		Message: message,
		Data:    data,
	})
}

func ErrorResponse(c *gin.Context, statusCode int, message string) {

	c.JSON(statusCode, Response{
		Status:  "Fail",
		Message: message,
	})
}

func mapErrorToStatus(err error) int {
	errMsg := strings.ToLower(err.Error())

	if strings.Contains(errMsg, "not found") {
		return http.StatusNotFound
	}
	if strings.Contains(errMsg, "already exists") || strings.Contains(errMsg, "duplicate key") || strings.Contains(errMsg, "unique constraint") {
		return http.StatusConflict // 409 Conflict often better for duplicates
	}
	if strings.Contains(errMsg, "invalid") || strings.Contains(errMsg, "cannot be empty") || strings.Contains(errMsg, "must be positive") || strings.Contains(errMsg, "cannot be negative") || strings.Contains(errMsg, "constraint violation") {
		return http.StatusBadRequest // 400 Bad Request for validation errors
	}
	if strings.Contains(errMsg, "does not exist") && (strings.Contains(errMsg, "category") || strings.Contains(errMsg, "foreign key")) {

		return http.StatusBadRequest // 400 Bad Request because the reference is invalid
	}

	return http.StatusInternalServerError
}
