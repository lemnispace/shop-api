package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lemnispace/shop-api/internal/services"
)

// AuthMiddleware creates a middleware that validates JWT tokens
func AuthMiddleware(authService services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Authorization header required",
				},
			})
			c.Abort()
			return
		}

		// Check for Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "INVALID_TOKEN",
					"message": "Invalid authorization header format. Expected: Bearer <token>",
				},
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "INVALID_TOKEN",
					"message": "Invalid or expired token",
				},
			})
			c.Abort()
			return
		}

		// Add customer info to context
		c.Set("customerID", claims.CustomerID)
		c.Set("customerEmail", claims.Email)

		c.Next()
	}
}

// OptionalAuthMiddleware creates a middleware that optionally validates JWT tokens
// If token is present, it validates it. If not present, request continues without auth.
func OptionalAuthMiddleware(authService services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		tokenString := parts[1]
		claims, err := authService.ValidateToken(tokenString)
		if err == nil {
			c.Set("customerID", claims.CustomerID)
			c.Set("customerEmail", claims.Email)
		}

		c.Next()
	}
}

// GetCustomerID extracts customer ID from gin context
func GetCustomerID(c *gin.Context) (string, bool) {
	customerID, exists := c.Get("customerID")
	if !exists {
		return "", false
	}
	id, ok := customerID.(string)
	return id, ok
}

// GetCustomerEmail extracts customer email from gin context
func GetCustomerEmail(c *gin.Context) (string, bool) {
	customerEmail, exists := c.Get("customerEmail")
	if !exists {
		return "", false
	}
	email, ok := customerEmail.(string)
	return email, ok
}
