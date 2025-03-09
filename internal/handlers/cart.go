package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/internal/services"
)

// CartService is the interface that handles cart operations
var cartService services.CartServiceInterface

// SetCartService sets the cart service to be used by the handlers
func SetCartService(service services.CartServiceInterface) {
	cartService = service
}

// cartErrorResponse provides a consistent error response structure for cart operations
func cartErrorResponse(w http.ResponseWriter, err error, defaultStatusCode int, defaultMessage string) {
	w.Header().Set("Content-Type", "application/json")

	statusCode := defaultStatusCode
	message := defaultMessage

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
		// Use the error message directly if it's not a known error
		if err != nil {
			message = err.Error()
		}
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    statusCode,
			"message": message,
		},
	})
}

// CartHandler handles all cart-related operations (create cart)
func CartHandler(w http.ResponseWriter, r *http.Request) {
	// If no cart service is available, return an error
	if cartService == nil {
		http.Error(w, "Cart service not available", http.StatusInternalServerError)
		return
	}

	// Check if this is a customer carts request
	query := r.URL.Query()
	customerID := query.Get("customer")

	if customerID != "" {
		// Handle customer carts request
		switch r.Method {
		case http.MethodGet:
			getCustomerCarts(w, r, customerID)
			return
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}

	// Regular cart operations
	switch r.Method {
	case http.MethodPost:
		createCart(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// CartDetailHandler handles operations on a specific cart (get, update)
func CartDetailHandler(w http.ResponseWriter, r *http.Request) {
	// If no cart service is available, return an error
	if cartService == nil {
		http.Error(w, "Cart service not available", http.StatusInternalServerError)
		return
	}

	// Extract cart ID from path
	path := strings.TrimPrefix(r.URL.Path, "/v1/cart/")
	segments := strings.Split(path, "/")

	// Handle cart checkout endpoint
	if len(segments) >= 2 && segments[1] == "checkout" {
		cartID := segments[0]
		if r.Method == http.MethodPost {
			getCartCheckout(w, r, cartID)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Handle cart items endpoints
	if len(segments) >= 2 && segments[1] == "items" {
		cartID := segments[0]
		if len(segments) == 2 {
			// /cart/{cartId}/items
			switch r.Method {
			case http.MethodPost:
				addCartItem(w, r, cartID)
				return
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
		} else if len(segments) == 3 {
			// /cart/{cartId}/items/{itemId}
			cartID := segments[0]
			itemID := segments[2]
			switch r.Method {
			case http.MethodPut:
				updateCartItem(w, r, cartID, itemID)
				return
			case http.MethodDelete:
				removeCartItem(w, r, cartID, itemID)
				return
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
		}
	}

	// Handle specific cart
	cartID := segments[0]
	switch r.Method {
	case http.MethodGet:
		getCart(w, r, cartID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// createCart handles the creation of a new shopping cart
func createCart(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var requestBody struct {
		CustomerID string `json:"customerId,omitempty"`
	}

	// Decode request body
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil && err != io.EOF {
		cartErrorResponse(w, nil, http.StatusBadRequest, "Invalid request format. Please provide a valid JSON body")
		return
	}

	// Create cart
	cart, err := cartService.CreateCart(r.Context(), requestBody.CustomerID)
	if err != nil {
		cartErrorResponse(w, err, http.StatusInternalServerError, "Failed to create cart")
		return
	}

	// Return cart in response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cart)
}

// getCart retrieves the details of a specific cart
func getCart(w http.ResponseWriter, r *http.Request, cartID string) {
	// Get cart
	cart, err := cartService.GetCart(r.Context(), cartID)
	if err != nil {
		cartErrorResponse(w, err, http.StatusInternalServerError, "Failed to retrieve cart")
		return
	}

	// Return cart in response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cart)
}

// addCartItem adds an item to a cart
func addCartItem(w http.ResponseWriter, r *http.Request, cartID string) {
	// Parse request body
	var input models.CartItemInput

	// Decode request body
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		cartErrorResponse(w, nil, http.StatusBadRequest, "Invalid request format. Please provide a valid JSON body")
		return
	}

	// Validate request
	if input.ProductID == "" {
		cartErrorResponse(w, nil, http.StatusBadRequest, "ProductID is required")
		return
	}

	if input.Quantity <= 0 {
		cartErrorResponse(w, nil, http.StatusBadRequest, "Quantity must be greater than 0")
		return
	}

	// Add item to cart
	cartItem, err := cartService.AddItem(r.Context(), cartID, input)
	if err != nil {
		cartErrorResponse(w, err, http.StatusInternalServerError, "Failed to add item to cart")
		return
	}

	// Return cart item in response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cartItem)
}

// updateCartItem updates an item in a cart
func updateCartItem(w http.ResponseWriter, r *http.Request, cartID string, itemID string) {
	// Parse request body
	var requestBody struct {
		Quantity int `json:"quantity"`
	}

	// Decode request body
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		cartErrorResponse(w, nil, http.StatusBadRequest, "Invalid request format. Please provide a valid JSON body")
		return
	}

	// Validate request
	if requestBody.Quantity <= 0 {
		cartErrorResponse(w, nil, http.StatusBadRequest, "Quantity must be greater than 0")
		return
	}

	// Update item in cart
	cartItem, err := cartService.UpdateItem(r.Context(), cartID, itemID, requestBody.Quantity)
	if err != nil {
		cartErrorResponse(w, err, http.StatusInternalServerError, "Failed to update item in cart")
		return
	}

	// Return updated cart item in response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cartItem)
}

// removeCartItem removes an item from a cart
func removeCartItem(w http.ResponseWriter, r *http.Request, cartID string, itemID string) {
	// Remove item from cart
	err := cartService.RemoveItem(r.Context(), cartID, itemID)
	if err != nil {
		cartErrorResponse(w, err, http.StatusInternalServerError, "Failed to remove item from cart")
		return
	}

	// Return no content
	w.WriteHeader(http.StatusNoContent)
}

// getCartCheckout retrieves a checkout URL for a cart
func getCartCheckout(w http.ResponseWriter, r *http.Request, cartID string) {
	// Get checkout URL
	checkoutResponse, err := cartService.GetCheckoutURL(r.Context(), cartID)
	if err != nil {
		cartErrorResponse(w, err, http.StatusInternalServerError, "Failed to generate checkout URL")
		return
	}

	// Return checkout URL in response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(checkoutResponse)
}

// getCustomerCarts retrieves all carts for a customer
func getCustomerCarts(w http.ResponseWriter, r *http.Request, customerID string) {
	// Parse query parameters
	query := r.URL.Query()
	includeExpiredParam := query.Get("includeExpired")
	includeExpired := includeExpiredParam == "true"

	// Get carts for customer
	carts, err := cartService.GetCartsByCustomer(r.Context(), customerID, includeExpired)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Format response
	response := struct {
		Carts []models.Cart `json:"carts"`
	}{
		Carts: make([]models.Cart, 0, len(carts)),
	}

	for _, cart := range carts {
		response.Carts = append(response.Carts, *cart)
	}

	// Return customer carts in response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
