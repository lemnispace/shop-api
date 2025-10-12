package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lemnispace/shop-api/internal/middleware"
	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/internal/services"
)

// Package-level service variables
var (
	authService     services.AuthService
	customerService services.CustomerService
)

// SetAuthService sets the auth service for handlers
func SetAuthService(service services.AuthService) {
	authService = service
}

// SetCustomerService sets the customer service for handlers
func SetCustomerService(service services.CustomerService) {
	customerService = service
}

// RegisterCustomer handles POST /v1/customers/register
func RegisterCustomer(c *gin.Context) {
	ctx := c.Request.Context()

	var input models.CustomerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
				"details": err.Error(),
			},
		})
		return
	}

	// Validate required fields
	if input.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "MISSING_FIELD",
				"message": "email is required",
			},
		})
		return
	}

	if input.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "MISSING_FIELD",
				"message": "password is required",
			},
		})
		return
	}

	if len(input.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_PASSWORD",
				"message": "password must be at least 8 characters",
			},
		})
		return
	}

	// Register customer
	response, err := authService.Register(ctx, &input)
	if err != nil {
		log.Printf("[ERROR] Failed to register customer: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "REGISTRATION_FAILED",
				"message": "Failed to register customer",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// LoginCustomer handles POST /v1/customers/login
func LoginCustomer(c *gin.Context) {
	ctx := c.Request.Context()

	var input models.CustomerLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
				"details": err.Error(),
			},
		})
		return
	}

	// Validate required fields
	if input.Email == "" || input.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "MISSING_CREDENTIALS",
				"message": "email and password are required",
			},
		})
		return
	}

	// Attempt login
	response, err := authService.Login(ctx, input.Email, input.Password)
	if err != nil {
		log.Printf("[WARN] Failed login attempt for email: %s", input.Email)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "INVALID_CREDENTIALS",
				"message": "Invalid email or password",
			},
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// RefreshToken handles POST /v1/customers/refresh
func RefreshToken(c *gin.Context) {
	var input struct {
		RefreshToken string `json:"refreshToken"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
				"details": err.Error(),
			},
		})
		return
	}

	if input.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "MISSING_FIELD",
				"message": "refreshToken is required",
			},
		})
		return
	}

	// Generate new tokens
	response, err := authService.RefreshToken(input.RefreshToken)
	if err != nil {
		log.Printf("[WARN] Failed to refresh token: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "INVALID_TOKEN",
				"message": "Invalid or expired refresh token",
			},
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetCustomerProfile handles GET /v1/customers/me
func GetCustomerProfile(c *gin.Context) {
	ctx := c.Request.Context()

	// Get customer ID from auth middleware
	customerID, exists := middleware.GetCustomerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "Authentication required",
			},
		})
		return
	}

	// Get customer
	customer, err := customerService.GetCustomer(ctx, customerID)
	if err != nil {
		log.Printf("[ERROR] Failed to get customer %s: %v", customerID, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "CUSTOMER_NOT_FOUND",
				"message": "Customer not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, customer)
}

// UpdateCustomerProfile handles PUT /v1/customers/me
func UpdateCustomerProfile(c *gin.Context) {
	ctx := c.Request.Context()

	// Get customer ID from auth middleware
	customerID, exists := middleware.GetCustomerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "Authentication required",
			},
		})
		return
	}

	var input models.CustomerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
				"details": err.Error(),
			},
		})
		return
	}

	// Update customer
	err := customerService.UpdateCustomer(ctx, customerID, &input)
	if err != nil {
		log.Printf("[ERROR] Failed to update customer %s: %v", customerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "UPDATE_FAILED",
				"message": "Failed to update customer profile",
				"details": err.Error(),
			},
		})
		return
	}

	// Return updated customer
	customer, err := customerService.GetCustomer(ctx, customerID)
	if err != nil {
		log.Printf("[ERROR] Failed to get updated customer %s: %v", customerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Profile updated but failed to retrieve",
			},
		})
		return
	}

	c.JSON(http.StatusOK, customer)
}

// DeleteCustomerAccount handles DELETE /v1/customers/me
func DeleteCustomerAccount(c *gin.Context) {
	ctx := c.Request.Context()

	// Get customer ID from auth middleware
	customerID, exists := middleware.GetCustomerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "Authentication required",
			},
		})
		return
	}

	// Delete customer
	err := customerService.DeleteCustomer(ctx, customerID)
	if err != nil {
		log.Printf("[ERROR] Failed to delete customer %s: %v", customerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "DELETE_FAILED",
				"message": "Failed to delete customer account",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Account deleted successfully",
	})
}
