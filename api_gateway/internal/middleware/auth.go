package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

func JWTMiddleware(secretKey string, log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Warn("Authorization header is missing")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			log.Warnf("Invalid Authorization header format: %s", authHeader)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
			return
		}

		tokenString := parts[1]
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secretKey), nil
		})

		if err != nil {
			log.Warnf("Failed to parse JWT token: %v", err)
			if errors.Is(err, jwt.ErrTokenExpired) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token expired"})
			} else if errors.Is(err, jwt.ErrTokenMalformed) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Malformed token"})
			} else if errors.Is(err, jwt.ErrTokenNotValidYet) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token not active yet"})
			} else {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			}
			return
		}

		if !token.Valid {
			log.Warn("Invalid JWT token provided")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		if claims.UserID <= 0 {
			log.Error("UserID missing or invalid in JWT claims")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Invalid token claims (missing user_id)"})
			return
		}

		c.Set("userID", claims.UserID)
		log.Infof("JWT authenticated successfully for UserID: %d", claims.UserID)

		c.Next()
	}
}

func GenerateTestToken(userID int, secretKey string) (string, error) {
	expirationTime := time.Now().Add(1 * time.Hour)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "api_gateway", // Пример
			Subject:   fmt.Sprintf("%d", userID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
