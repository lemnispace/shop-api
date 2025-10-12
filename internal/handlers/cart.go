package handlers

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/internal/services"
	"github.com/lemnispace/shop-api/internal/utils"
)

// CartService is the interface that handles cart operations
var cartService services.CartServiceInterface

// SetCartService sets the cart service to be used by the handlers
func SetCartService(service services.CartServiceInterface) {
	cartService = service
}

// cartErrorResponse provides a consistent error response structure for cart operations
func cartErrorResponse(c *gin.Context, err error, defaultStatusCode int, defaultMessage string) {
	statusCode := defaultStatusCode
	message := defaultMessage
	// errorCode determination is handled by utils.ErrorResponseWithDetails

	switch err {
	case services.ErrCartNotFound:
		statusCode = http.StatusNotFound
		message = "Cart not found or has been deleted"
	case services.ErrCartExpired:
		statusCode = http.StatusGone
		message = "Cart has expired. Please create a new cart"
	case services.ErrProductNotFound:
		statusCode = http.StatusNotFound
		message = "The requested product could not be found"
	case services.ErrVariantNotFound:
		statusCode = http.StatusNotFound
		message = "The requested product variant could not be found"
	case services.ErrInsufficientInventory:
		statusCode = http.StatusBadRequest
		message = "Product does not have sufficient inventory for the requested quantity"
	case services.ErrItemNotInCart:
		statusCode = http.StatusNotFound
		message = "The requested item is not in the cart"
	case services.ErrInvalidQuantity:
		statusCode = http.StatusBadRequest
		message = "Quantity must be a positive number"
	default:
		if err != nil {
			message = err.Error()
		}
	}

	utils.ErrorResponseWithDetails(c, statusCode, message, nil)
}

// CreateCart handles POST /v1/cart
func CreateCart(c *gin.Context) {
	if cartService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Cart service not available")
		return
	}

	var requestBody struct {
		CustomerID string `json:"customerId,omitempty"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil && err != io.EOF {
		cartErrorResponse(c, err, http.StatusBadRequest, "Invalid request format")
		return
	}

	cart, err := cartService.CreateCart(c.Request.Context(), requestBody.CustomerID)
	if err != nil {
		cartErrorResponse(c, err, http.StatusInternalServerError, "Failed to create cart")
		return
	}

	utils.JSONResponse(c, http.StatusCreated, cart)
}

// GetCustomerCarts handles GET /v1/cart?customer={customerId}
func GetCustomerCarts(c *gin.Context) {
	panic("not implemented")
}

// GetCart handles GET /v1/cart/:cartId
func GetCart(c *gin.Context) {
	if cartService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Cart service not available")
		return
	}

	cartID := c.Param("cartId")
	if cartID == "" {
		cartErrorResponse(c, nil, http.StatusBadRequest, "Cart ID is required")
		return
	}

	cart, err := cartService.GetCart(c.Request.Context(), cartID)
	if err != nil {
		cartErrorResponse(c, err, http.StatusInternalServerError, "Failed to retrieve cart")
		return
	}

	utils.JSONResponse(c, http.StatusOK, cart)
}

// AddCartItem handles POST /v1/cart/:cartId/items
func AddCartItem(c *gin.Context) {
	if cartService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Cart service not available")
		return
	}

	cartID := c.Param("cartId")
	if cartID == "" {
		cartErrorResponse(c, nil, http.StatusBadRequest, "Cart ID is required")
		return
	}

	var input models.CartItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		cartErrorResponse(c, err, http.StatusBadRequest, "Invalid request format")
		return
	}

	if input.ProductID == "" {
		cartErrorResponse(c, nil, http.StatusBadRequest, "ProductID is required")
		return
	}

	if input.Quantity <= 0 {
		cartErrorResponse(c, services.ErrInvalidQuantity, http.StatusBadRequest, "Quantity must be greater than 0")
		return
	}

	cartItem, err := cartService.AddItem(c.Request.Context(), cartID, input)
	if err != nil {
		cartErrorResponse(c, err, http.StatusInternalServerError, "Failed to add item to cart")
		return
	}

	utils.JSONResponse(c, http.StatusOK, cartItem)
}

// UpdateCartItem handles PUT /v1/cart/:cartId/items/:itemId
func UpdateCartItem(c *gin.Context) {
	if cartService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Cart service not available")
		return
	}

	cartID := c.Param("cartId")
	itemID := c.Param("itemId")
	if cartID == "" || itemID == "" {
		cartErrorResponse(c, nil, http.StatusBadRequest, "Cart ID and Item ID are required")
		return
	}

	var requestBody struct {
		Quantity int `json:"quantity" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		cartErrorResponse(c, err, http.StatusBadRequest, "Invalid request format or quantity must be positive")
		return
	}

	// Assuming UpdateItem exists and returns the updated item
	updatedItem, err := cartService.UpdateItem(c.Request.Context(), cartID, itemID, requestBody.Quantity)
	if err != nil {
		cartErrorResponse(c, err, http.StatusInternalServerError, "Failed to update cart item quantity")
		return
	}

	utils.JSONResponse(c, http.StatusOK, updatedItem)
}

// RemoveCartItem handles DELETE /v1/cart/:cartId/items/:itemId
func RemoveCartItem(c *gin.Context) {
	if cartService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Cart service not available")
		return
	}

	cartID := c.Param("cartId")
	itemID := c.Param("itemId")
	if cartID == "" || itemID == "" {
		cartErrorResponse(c, nil, http.StatusBadRequest, "Cart ID and Item ID are required")
		return
	}

	err := cartService.RemoveItem(c.Request.Context(), cartID, itemID)
	if err != nil {
		cartErrorResponse(c, err, http.StatusInternalServerError, "Failed to remove item from cart")
		return
	}

	utils.NoContent(c)
}

// GetCartCheckout handles POST /v1/cart/:cartId/checkout
func GetCartCheckout(c *gin.Context) {
	if cartService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Cart service not available")
		return
	}

	cartID := c.Param("cartId")
	if cartID == "" {
		cartErrorResponse(c, nil, http.StatusBadRequest, "Cart ID is required")
		return
	}

	checkoutResponse, err := cartService.GetCheckoutURL(c.Request.Context(), cartID)
	if err != nil {
		cartErrorResponse(c, err, http.StatusInternalServerError, "Failed to generate checkout URL")
		return
	}

	utils.JSONResponse(c, http.StatusOK, checkoutResponse)
}
