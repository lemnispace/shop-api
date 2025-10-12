package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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

	switch err {
	case services.ErrOrderNotFound:
		statusCode = http.StatusNotFound
		message = "Order not found"
	case services.ErrCartNotFound:
		statusCode = http.StatusNotFound
		message = "Cart not found"
	case services.ErrCartEmpty:
		statusCode = http.StatusBadRequest
		message = "Cart is empty - cannot create order"
	case services.ErrInvalidOrderStatus:
		statusCode = http.StatusBadRequest
		message = "Invalid order status"
	default:
		if err != nil {
			message = err.Error()
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

	if input.CustomerID == "" {
		orderErrorResponse(c, nil, http.StatusBadRequest, "Customer ID is required")
		return
	}

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

	utils.JSONResponse(c, http.StatusOK, order)
}

// ListOrders handles GET /v1/orders
func ListOrders(c *gin.Context) {
	if orderService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Order service not available")
		return
	}

	// Check if filtering by customer
	customerID := c.Query("customerId")
	if customerID != "" {
		listOrdersByCustomer(c, customerID)
		return
	}

	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	cursor := c.Query("cursor")

	// Get filters from query params
	filters := make(map[string]interface{})
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}

	result, err := orderService.ListOrders(c.Request.Context(), limit, cursor, filters)
	if err != nil {
		orderErrorResponse(c, err, http.StatusInternalServerError, "Failed to list orders")
		return
	}

	utils.JSONResponse(c, http.StatusOK, result)
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
func UpdateOrderStatus(c *gin.Context) {
	if orderService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Order service not available")
		return
	}

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
}

// CancelOrder handles POST /v1/orders/:orderId/cancel
func CancelOrder(c *gin.Context) {
	if orderService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Order service not available")
		return
	}

	orderID := c.Param("orderId")
	if orderID == "" {
		orderErrorResponse(c, nil, http.StatusBadRequest, "Order ID is required")
		return
	}

	err := orderService.CancelOrder(c.Request.Context(), orderID)
	if err != nil {
		orderErrorResponse(c, err, http.StatusInternalServerError, "Failed to cancel order")
		return
	}

	// Return updated order
	order, err := orderService.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		orderErrorResponse(c, err, http.StatusInternalServerError, "Failed to retrieve cancelled order")
		return
	}

	utils.JSONResponse(c, http.StatusOK, order)
}
