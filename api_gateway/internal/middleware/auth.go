package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func AuthMiddleware(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Warn("Middleware: Authorization header is missing")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			log.Warnf("Middleware: Invalid Authorization header format: %s", authHeader)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
			return
		}

		rawToken := parts[1]
		if rawToken == "" {
			log.Warn("Middleware: Bearer token is empty")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		log.Debugf("Middleware: Extracted raw token (UUID): %s...", rawToken[:min(10, len(rawToken))])

		c.Set("rawToken", rawToken)
		c.Next()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func RequestLogger(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		entry := logger.WithFields(logrus.Fields{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"remote_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		})
		if reqID := c.Writer.Header().Get("X-Request-ID"); reqID != "" {
			entry = entry.WithField("request_id", reqID)
		}
		entry.Info("Incoming request")

		c.Next()

		latency := time.Since(startTime)
		statusCode := c.Writer.Status()

		completedEntry := logger.WithFields(logrus.Fields{
			"status_code": statusCode,
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"remote_ip":   c.ClientIP(),
			"latency_ms":  latency.Milliseconds(),
		})
		if reqID := c.Writer.Header().Get("X-Request-ID"); reqID != "" {
			completedEntry = completedEntry.WithField("request_id", reqID)
		}

		if len(c.Errors) > 0 {
			completedEntry.Error(c.Errors.ByType(gin.ErrorTypePrivate).String())
		} else {
			if statusCode >= 500 {
				completedEntry.Error("Request completed with server error")
			} else if statusCode >= 400 {
				completedEntry.Warn("Request completed with client error")
			} else {
				completedEntry.Info("Request completed successfully")
			}
		}
	}
}
