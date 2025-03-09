package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lemnispace/shop-api/internal/handlers"
	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/internal/services"
	"github.com/lemnispace/shop-api/tests"
	"github.com/stretchr/testify/assert"
)

// TestCartSingleTableDesignAPI verifies that the Cart API works correctly with the single table design
func TestCartSingleTableDesignAPI(t *testing.T) {
	// Set up test environment
	router, _, cleanup := setupCartTestEnvironment(t)
	defer cleanup()

	// Create cart
	cart := createTestCart(t, router)
	assert.NotEmpty(t, cart.ID, "Cart ID should not be empty")

	// Verify we can retrieve the cart
	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/cart/%s", cart.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK")

	// Verify key format directly
	pk, sk := services.CartKey(cart.ID)
	expectedPK := fmt.Sprintf("%s#%s", services.EntityCart, cart.ID)
	expectedSK := fmt.Sprintf("%s#%s", services.EntityCart, cart.ID)

	assert.Equal(t, expectedPK, pk, "Cart PK format doesn't match expected pattern")
	assert.Equal(t, expectedSK, sk, "Cart SK format doesn't match expected pattern")

	// Add an item to the cart
	addItemToCart(t, router, cart.ID)

	// Get the cart again to verify item was added
	req = httptest.NewRequest("GET", fmt.Sprintf("/v1/cart/%s", cart.ID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK")

	// Parse the cart with item
	var updatedCart models.Cart
	err := json.NewDecoder(w.Body).Decode(&updatedCart)
	assert.NoError(t, err, "Failed to decode cart JSON")

	// Verify the cart contains the item
	assert.NotEmpty(t, updatedCart.Items, "Cart should have items")
}

// Helper function to set up cart test environment
func setupCartTestEnvironment(t *testing.T) (http.Handler, *services.CartService, func()) {
	// Set up DynamoDB client
	client, err := tests.SetupDynamoDBForTesting()
	if err != nil {
		t.Fatalf("Failed to set up DynamoDB for testing: %v", err)
	}

	// Set up router
	router := http.NewServeMux()

	// Set up services
	productService := services.NewProductService(client, tests.TestTableName())
	handlers.SetProductService(productService)

	collectionService := services.NewCollectionService(client, tests.TestTableName(), productService)
	handlers.SetCollectionService(collectionService)

	// Create a product for cart items
	product := &models.Product{
		Title:       "Test Product",
		Description: "A product for testing",
		Price:       29.99,
		SKU:         "TEST-SKU-1",
		Status:      "active",
		Inventory:   100,
	}

	err = productService.CreateProduct(context.Background(), product)
	if err != nil {
		t.Fatalf("Failed to create test product: %v", err)
	}

	// Create cart service
	cartService := services.NewCartService(client, productService, tests.TestTableName())
	handlers.SetCartService(cartService)

	// Set up all required routes
	// Cart routes
	router.HandleFunc("/v1/cart", handlers.CartHandler)
	router.HandleFunc("/v1/cart/", handlers.CartDetailHandler)

	// Product routes - needed for createTestProduct function
	router.HandleFunc("/v1/products", handlers.ProductsHandler)
	router.HandleFunc("/v1/products/", handlers.ProductDetailHandler)
	router.HandleFunc("/v1/products/count", handlers.ProductCountHandler)
	router.HandleFunc("/v1/products/variants", handlers.ProductVariantsHandler)

	// Cleanup function
	cleanup := func() {
		if err := tests.TeardownDynamoDBForTesting(client); err != nil {
			t.Logf("Warning: Failed to tear down DynamoDB: %v", err)
		}
	}

	return router, cartService, cleanup
}

// Helper function to create a test cart
func createTestCart(t *testing.T, handler http.Handler) *models.Cart {
	// Create request body for cart creation
	body := map[string]string{
		"customerId": "cust123",
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal cart data: %v", err)
	}

	// Create request
	req := httptest.NewRequest("POST", "/v1/cart", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Send request
	handler.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusCreated, w.Code, "Expected 201 Created")

	// Parse response
	var cart models.Cart
	err = json.NewDecoder(w.Body).Decode(&cart)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	return &cart
}

// Helper function to add an item to a cart
func addItemToCart(t *testing.T, handler http.Handler, cartID string) *models.CartItem {
	// Create a test product directly instead of trying to get an existing one
	product := createTestProduct(t, handler)

	// Create cart item input
	input := models.CartItemInput{
		ProductID: product.ID,
		Quantity:  2,
	}

	// Add variant ID if available
	if len(product.Variants) > 0 {
		input.VariantID = product.Variants[0].ID
	}

	// Convert to JSON
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal cart item data: %v", err)
	}

	// Create request
	req := httptest.NewRequest("POST", fmt.Sprintf("/v1/cart/%s/items", cartID), bytes.NewReader(inputJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Send request
	handler.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK")

	// Parse response
	var cartItem models.CartItem
	err = json.NewDecoder(w.Body).Decode(&cartItem)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	return &cartItem
}
