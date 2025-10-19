package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lemnispace/shop-api/internal/middleware"
	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/internal/services"
	"github.com/lemnispace/shop-api/internal/utils"
)

// OrderService is the interface that handles order operations
var orderService services.OrderService

// SetOrderService sets the order service to be used by the handlers
func SetOrderService(service services.OrderService) {
	orderService = service
}

// orderErrorResponse provides a consistent error response structure for order operations
func orderErrorResponse(c *gin.Context, err error, defaultStatusCode int, defaultMessage string) {
	statusCode := defaultStatusCode
	message := defaultMessage

	// Use errors.Is to check for wrapped errors
	switch {
	case errors.Is(err, services.ErrOrderNotFound):
		statusCode = http.StatusNotFound
		message = "Order not found"
	case errors.Is(err, services.ErrCartNotFound):
		statusCode = http.StatusNotFound
		message = "Cart not found"
	case errors.Is(err, services.ErrCartEmpty):
		statusCode = http.StatusBadRequest
		message = "Cart is empty - cannot create order"
	case errors.Is(err, services.ErrCartExpired):
		statusCode = http.StatusGone
		message = "Cart has expired - please create a new cart"
	case errors.Is(err, services.ErrInvalidOrderStatus):
		statusCode = http.StatusBadRequest
		message = "Invalid order status"
	default:
		if err != nil {
			// Don't leak internal error details - use default message
			message = defaultMessage
		}
	}

	utils.ErrorResponseWithDetails(c, statusCode, message, nil)
}

// CreateOrder handles POST /v1/orders
func CreateOrder(c *gin.Context) {
	if orderService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Order service not available")
		return
	}

	// Get authenticated customer ID
	authenticatedCustomerID, exists := middleware.GetCustomerID(c)
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Authentication required")
		return
	}

	var input models.OrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		orderErrorResponse(c, err, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate required fields
	if input.CartID == "" {
		orderErrorResponse(c, nil, http.StatusBadRequest, "Cart ID is required")
		return
	}

	// SECURITY: Use authenticated customer ID, not client-supplied value
	// This prevents users from creating orders for other customers
	input.CustomerID = authenticatedCustomerID

	// Create order
	order, err := orderService.CreateOrder(c.Request.Context(), &input)
	if err != nil {
		orderErrorResponse(c, err, http.StatusInternalServerError, "Failed to create order")
		return
	}

	utils.JSONResponse(c, http.StatusCreated, order)
}

// GetOrder handles GET /v1/orders/:orderId
func GetOrder(c *gin.Context) {
	if orderService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Order service not available")
		return
	}

	// Get authenticated customer ID
	authenticatedCustomerID, exists := middleware.GetCustomerID(c)
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Authentication required")
		return
	}

	orderID := c.Param("orderId")
	if orderID == "" {
		orderErrorResponse(c, nil, http.StatusBadRequest, "Order ID is required")
		return
	}

	order, err := orderService.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		orderErrorResponse(c, err, http.StatusInternalServerError, "Failed to retrieve order")
		return
	}

	// SECURITY: Verify order belongs to authenticated customer
	// TODO: Add admin role check to allow admins to view any order
	if order.CustomerID != authenticatedCustomerID {
		utils.ErrorResponse(c, http.StatusForbidden, "Access denied - order belongs to another customer")
		return
	}

	utils.JSONResponse(c, http.StatusOK, order)
}

// ListOrders handles GET /v1/orders
func ListOrders(c *gin.Context) {
	if orderService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Order service not available")
		return
	}

	// Get authenticated customer ID
	authenticatedCustomerID, exists := middleware.GetCustomerID(c)
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Check if filtering by customer
	requestedCustomerID := c.Query("customerId")

	// SECURITY: Users can only list their own orders unless they're admin
	// TODO: Add admin role check to allow admins to query any customer's orders
	if requestedCustomerID != "" && requestedCustomerID != authenticatedCustomerID {
		utils.ErrorResponse(c, http.StatusForbidden, "Access denied - cannot list orders for other customers")
		return
	}

	// If no specific customer requested, use authenticated customer
	if requestedCustomerID == "" {
		requestedCustomerID = authenticatedCustomerID
	}

	listOrdersByCustomer(c, requestedCustomerID)
}

// listOrdersByCustomer is a helper function to list orders by customer
func listOrdersByCustomer(c *gin.Context, customerID string) {
	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	cursor := c.Query("cursor")

	result, err := orderService.GetOrdersByCustomer(c.Request.Context(), customerID, limit, cursor)
	if err != nil {
		orderErrorResponse(c, err, http.StatusInternalServerError, "Failed to list customer orders")
		return
	}

	utils.JSONResponse(c, http.StatusOK, result)
}

// UpdateOrderStatus handles PATCH /v1/orders/:orderId
// This is an admin-only operation
func UpdateOrderStatus(c *gin.Context) {
	if orderService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Order service not available")
		return
	}

	// Get authenticated customer ID
	_, exists := middleware.GetCustomerID(c)
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Authentication required")
		return
	}

	// TODO: Add admin role check here
	// For now, this endpoint should only be accessible to admins via future role-based middleware
	// Until Customer model supports roles, we disable arbitrary status updates
	utils.ErrorResponse(c, http.StatusForbidden, "Admin access required")
	return

	// The code below would execute once admin roles are implemented:
	/*
	orderID := c.Param("orderId")
	if orderID == "" {
		orderErrorResponse(c, nil, http.StatusBadRequest, "Order ID is required")
		return
	}

	var requestBody struct {
		Status models.OrderStatus `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		orderErrorResponse(c, err, http.StatusBadRequest, "Invalid request format")
		return
	}

	err := orderService.UpdateOrderStatus(c.Request.Context(), orderID, requestBody.Status)
	if err != nil {
		orderErrorResponse(c, err, http.StatusInternalServerError, "Failed to update order status")
		return
	}

	// Return updated order
	order, err := orderService.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		orderErrorResponse(c, err, http.StatusInternalServerError, "Failed to retrieve updated order")
		return
	}

	utils.JSONResponse(c, http.StatusOK, order)
	*/
}

// CancelOrder handles POST /v1/orders/:orderId/cancel
func CancelOrder(c *gin.Context) {
	if orderService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Order service not available")
		return
	}

	// Get authenticated customer ID
	authenticatedCustomerID, exists := middleware.GetCustomerID(c)
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Authentication required")
		return
	}

	orderID := c.Param("orderId")
	if orderID == "" {
		orderErrorResponse(c, nil, http.StatusBadRequest, "Order ID is required")
		return
	}

	// SECURITY: Verify order belongs to authenticated customer before allowing cancellation
	order, err := orderService.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		orderErrorResponse(c, err, http.StatusNotFound, "Order not found")
		return
	}

	// TODO: Add admin role check to allow admins to cancel any order
	if order.CustomerID != authenticatedCustomerID {
		utils.ErrorResponse(c, http.StatusForbidden, "Access denied - cannot cancel another customer's order")
		return
	}

	err = orderService.CancelOrder(c.Request.Context(), orderID)
	if err != nil {
		orderErrorResponse(c, err, http.StatusInternalServerError, "Failed to cancel order")
		return
	}

	// Return updated order
	order, err = orderService.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		orderErrorResponse(c, err, http.StatusInternalServerError, "Failed to retrieve cancelled order")
		return
	}

	utils.JSONResponse(c, http.StatusOK, order)
}
