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
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create cart
	cart, err := cartService.CreateCart(r.Context(), requestBody.CustomerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		switch err {
		case services.ErrCartNotFound:
			http.Error(w, "Cart not found", http.StatusNotFound)
		case services.ErrCartExpired:
			http.Error(w, "Cart has expired", http.StatusGone)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if input.ProductID == "" {
		http.Error(w, "ProductID is required", http.StatusBadRequest)
		return
	}

	if input.Quantity <= 0 {
		http.Error(w, "Quantity must be greater than 0", http.StatusBadRequest)
		return
	}

	// Add item to cart
	cartItem, err := cartService.AddItem(r.Context(), cartID, input)
	if err != nil {
		switch err {
		case services.ErrCartNotFound:
			http.Error(w, "Cart not found", http.StatusNotFound)
		case services.ErrCartExpired:
			http.Error(w, "Cart has expired", http.StatusGone)
		case services.ErrProductNotFound:
			http.Error(w, "Product not found", http.StatusNotFound)
		case services.ErrVariantNotFound:
			http.Error(w, "Variant not found", http.StatusNotFound)
		case services.ErrProductNotInStock:
			http.Error(w, "Product not in stock", http.StatusConflict)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Return cart item in response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(cartItem)
}

// updateCartItem updates an item in a cart (typically quantity)
func updateCartItem(w http.ResponseWriter, r *http.Request, cartID string, itemID string) {
	// Parse request body
	var requestBody struct {
		Quantity int `json:"quantity"`
	}

	// Decode request body
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate quantity
	if requestBody.Quantity <= 0 {
		http.Error(w, "Quantity must be greater than 0", http.StatusBadRequest)
		return
	}

	// Update cart item
	cartItem, err := cartService.UpdateItem(r.Context(), cartID, itemID, requestBody.Quantity)
	if err != nil {
		switch err {
		case services.ErrCartNotFound:
			http.Error(w, "Cart not found", http.StatusNotFound)
		case services.ErrCartExpired:
			http.Error(w, "Cart has expired", http.StatusGone)
		case services.ErrCartItemNotFound:
			http.Error(w, "Cart item not found", http.StatusNotFound)
		case services.ErrProductNotInStock:
			http.Error(w, "Product not in stock", http.StatusConflict)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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
		switch err {
		case services.ErrCartNotFound:
			http.Error(w, "Cart not found", http.StatusNotFound)
		case services.ErrCartExpired:
			http.Error(w, "Cart has expired", http.StatusGone)
		case services.ErrCartItemNotFound:
			http.Error(w, "Cart item not found", http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Return no content for successful deletion
	w.WriteHeader(http.StatusNoContent)
}

// getCartCheckout generates a checkout URL for the cart
func getCartCheckout(w http.ResponseWriter, r *http.Request, cartID string) {
	// Get checkout URL
	checkoutResponse, err := cartService.GetCheckoutURL(r.Context(), cartID)
	if err != nil {
		switch err {
		case services.ErrCartNotFound:
			http.Error(w, "Cart not found", http.StatusNotFound)
		case services.ErrCartExpired:
			http.Error(w, "Cart has expired", http.StatusGone)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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
