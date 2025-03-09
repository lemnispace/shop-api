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

// TestCartMultipleProducts tests the cart API with multiple products
func TestCartMultipleProducts(t *testing.T) {
	// Set up test environment
	router, _, cleanup := setupCartTestEnvironment(t)
	defer cleanup()

	// Create cart
	cart := createTestCart(t, router)
	assert.NotEmpty(t, cart.ID, "Cart ID should not be empty")

	// Create first product
	firstProduct := createTestProduct(t, router)
	assert.NotEmpty(t, firstProduct.ID, "First product ID should not be empty")

	// Create second product with different price
	secondProduct := createTestProduct(t, router)
	assert.NotEmpty(t, secondProduct.ID, "Second product ID should not be empty")

	// Update second product to have a different price
	updateProductBody := fmt.Sprintf(`{
		"title": "Second Test Product",
		"description": "A second product for testing",
		"price": 49.99,
		"sku": "TEST-SKU-2",
		"status": "active",
		"inventory": 100
	}`)

	req := httptest.NewRequest("PUT", fmt.Sprintf("/v1/products/%s", secondProduct.ID), bytes.NewReader([]byte(updateProductBody)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK when updating second product")

	// Add first product to cart
	firstItem := models.CartItemInput{
		ProductID: firstProduct.ID,
		Quantity:  1,
	}
	firstItemJSON, err := json.Marshal(firstItem)
	assert.NoError(t, err, "Failed to marshal first cart item data")

	req = httptest.NewRequest("POST", fmt.Sprintf("/v1/cart/%s/items", cart.ID), bytes.NewReader(firstItemJSON))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK when adding first item")

	// Add second product to cart with quantity 3
	secondItem := models.CartItemInput{
		ProductID: secondProduct.ID,
		Quantity:  3,
	}
	secondItemJSON, err := json.Marshal(secondItem)
	assert.NoError(t, err, "Failed to marshal second cart item data")

	req = httptest.NewRequest("POST", fmt.Sprintf("/v1/cart/%s/items", cart.ID), bytes.NewReader(secondItemJSON))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK when adding second item")

	// Get the cart to check totals
	req = httptest.NewRequest("GET", fmt.Sprintf("/v1/cart/%s", cart.ID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK when getting cart")

	// Parse the cart
	var updatedCart models.Cart
	err = json.NewDecoder(w.Body).Decode(&updatedCart)
	assert.NoError(t, err, "Failed to decode cart JSON")

	// Verify the cart contains both items
	assert.Equal(t, 2, len(updatedCart.Items), "Cart should have 2 items")

	// Check items and quantities
	var firstItemFound, secondItemFound bool
	for _, item := range updatedCart.Items {
		if item.ProductID == firstProduct.ID {
			assert.Equal(t, 1, item.Quantity, "First item should have quantity 1")
			assert.Equal(t, 29.99, item.Price, "First item should have price 29.99")
			firstItemFound = true
		} else if item.ProductID == secondProduct.ID {
			assert.Equal(t, 3, item.Quantity, "Second item should have quantity 3")
			assert.Equal(t, 49.99, item.Price, "Second item should have price 49.99")
			secondItemFound = true
		}
	}
	assert.True(t, firstItemFound, "First item should be in cart")
	assert.True(t, secondItemFound, "Second item should be in cart")

	// Calculate and verify the expected subtotal: 1*29.99 + 3*49.99 = 179.96
	expectedSubtotal := 29.99 + (3 * 49.99)
	assert.InDelta(t, expectedSubtotal, updatedCart.Subtotal, 0.01, "Subtotal should be calculated correctly")

	// Calculate and verify the expected estimated tax (assuming 9% tax rate)
	expectedTax := expectedSubtotal * 0.09
	assert.InDelta(t, expectedTax, updatedCart.EstimatedTax, 0.01, "Estimated tax should be calculated correctly")

	// Verify total price (subtotal + tax + shipping)
	expectedTotal := expectedSubtotal + expectedTax + updatedCart.EstimatedShipping
	assert.InDelta(t, expectedTotal, updatedCart.TotalPrice, 0.01, "Total price should be calculated correctly")

	// Test removing one item
	firstItemID := ""
	for _, item := range updatedCart.Items {
		if item.ProductID == firstProduct.ID {
			firstItemID = item.ID
			break
		}
	}

	req = httptest.NewRequest("DELETE", fmt.Sprintf("/v1/cart/%s/items/%s", cart.ID, firstItemID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code, "Expected 204 No Content when removing an item")

	// Get the cart again to verify the item was removed
	req = httptest.NewRequest("GET", fmt.Sprintf("/v1/cart/%s", cart.ID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK when getting cart after item removal")

	// Parse the cart again
	var finalCart models.Cart
	err = json.NewDecoder(w.Body).Decode(&finalCart)
	assert.NoError(t, err, "Failed to decode final cart JSON")

	// Verify the cart now only contains the second item
	assert.Equal(t, 1, len(finalCart.Items), "Cart should have 1 item after removal")
	assert.Equal(t, secondProduct.ID, finalCart.Items[0].ProductID, "Remaining item should be the second product")

	// Verify the subtotal is updated correctly (3 * 49.99 = 149.97)
	expectedSubtotal = 3 * 49.99
	assert.InDelta(t, expectedSubtotal, finalCart.Subtotal, 0.01, "Updated subtotal should be calculated correctly")
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
