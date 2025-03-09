package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lemnispace/shop-api/internal/handlers"
	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/internal/services"
	"github.com/lemnispace/shop-api/tests"
	"github.com/stretchr/testify/assert"
)

// Helper function to set up test environment
func setupTestEnvironment(t *testing.T) (http.Handler, func()) {
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

	// Set up routes
	router.HandleFunc(apiPrefix+"/products", handlers.ProductsHandler)
	router.HandleFunc(apiPrefix+"/products/", handlers.ProductDetailHandler)
	router.HandleFunc(apiPrefix+"/products/count", handlers.ProductCountHandler)
	router.HandleFunc(apiPrefix+"/products/variants", handlers.ProductVariantsHandler)

	// Cleanup function
	cleanup := func() {
		if err := tests.TeardownDynamoDBForTesting(client); err != nil {
			t.Logf("Warning: Failed to tear down DynamoDB: %v", err)
		}
	}

	return router, cleanup
}

// Helper function to create a test product
func createTestProduct(t *testing.T, handler http.Handler) *models.Product {
	// Create product data
	product := models.Product{
		Title:       "Test Product",
		Description: "A product for testing",
		Price:       29.99,
		SKU:         "TEST-SKU-1",
		Status:      "active",
		Inventory:   100,
	}

	// Convert to JSON
	productJSON, err := json.Marshal(product)
	if err != nil {
		t.Fatalf("Failed to marshal product data: %v", err)
	}

	// Create request
	req := httptest.NewRequest("POST", "/v1/products", strings.NewReader(string(productJSON)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Send request
	handler.ServeHTTP(w, req)

	// Check status code
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	// Parse response
	var createdProduct models.Product
	if err := json.NewDecoder(w.Body).Decode(&createdProduct); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	return &createdProduct
}

// TestGetProducts is an integration test for the GET /products endpoint.
// Note: This test may require retry logic to handle DynamoDB eventual consistency.
func TestGetProducts(t *testing.T) {
	// Create a product first to ensure we have data
	newProduct := map[string]interface{}{
		"title":       "Test Product for GetProducts",
		"description": "A product created for get products test",
		"price":       29.99,
		"sku":         "TEST-SKU-GET-001",
		"status":      "active",
		"inventory":   50,
	}

	// Make request to create product
	createResp, createBody := MakeRequest(t, http.MethodPost, apiPrefix+"/products", newProduct)
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create test product: %d - %s", createResp.StatusCode, createBody)
	}

	// Parse the response to get the product ID
	var createdProduct models.Product
	err := json.Unmarshal([]byte(createBody), &createdProduct)
	if err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	t.Logf("Created product with ID: %s", createdProduct.ID)

	// Retry logic for eventual consistency
	var response struct {
		Items []models.Product       `json:"items"`
		Links models.PaginationLinks `json:"links"`
	}

	// Retry up to 5 times with increasing delays
	maxRetries := 5
	found := false

	for attempt := 0; attempt < maxRetries && !found; attempt++ {
		// Add a delay increasing with each attempt
		time.Sleep(time.Duration(100+attempt*100) * time.Millisecond)

		// Make request to list all products
		resp, body := MakeRequest(t, http.MethodGet, apiPrefix+"/products", nil)

		// Check status code
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}

		// Parse response
		err = json.Unmarshal([]byte(body), &response)
		if err != nil {
			t.Fatalf("Failed to parse response body: %v", err)
		}

		// If we have products and at least one matches our created ID, we're good
		if len(response.Items) > 0 {
			for _, p := range response.Items {
				if p.ID == createdProduct.ID {
					found = true
					break
				}
			}
		}

		// If we found what we needed, break out
		if found {
			break
		}

		t.Logf("Attempt %d: No matching products found yet, retrying...", attempt+1)
	}

	// Verify we have found our product
	if !found {
		t.Error("Expected to find the created product in the list, but it was not found")
	}

	// Check that the self link is set
	if response.Links.Self == "" {
		t.Error("Expected self link, got none")
	}
}

// TestGetProductByID tests the GET /products/{id} endpoint
func TestGetProductByID(t *testing.T) {
	// Create a product first to ensure we have data
	newProduct := map[string]interface{}{
		"title":       "Test Product for GetProductByID",
		"description": "A product created for get by ID test",
		"price":       19.99,
		"sku":         "TEST-SKU-GETID-001",
		"status":      "active",
		"inventory":   75,
	}

	// Make request to create product
	createResp, createBody := MakeRequest(t, http.MethodPost, apiPrefix+"/products", newProduct)
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create test product: %d - %s", createResp.StatusCode, createBody)
	}

	// Parse the created product to get its ID
	var createdProduct models.Product
	ParseJSONResponse(t, createBody, &createdProduct)

	if createdProduct.ID == "" {
		t.Fatal("Created product doesn't have an ID")
	}

	// Test getting a product by ID
	resp, body := MakeRequest(t, http.MethodGet, fmt.Sprintf("%s/products/%s", apiPrefix, createdProduct.ID), nil)

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Parse response
	var product models.Product
	ParseJSONResponse(t, body, &product)

	// Verify we got the right product
	if product.ID != createdProduct.ID {
		t.Errorf("Expected product ID %s, got %s", createdProduct.ID, product.ID)
	}

	// Test getting a non-existent product
	resp, _ = MakeRequest(t, http.MethodGet, fmt.Sprintf("%s/products/non-existent-id", apiPrefix), nil)

	// Check status code (should be 404)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

// TestCreateProduct tests the POST /products endpoint
func TestCreateProduct(t *testing.T) {
	// Create a new product
	newProduct := map[string]interface{}{
		"title":       "Test Product",
		"description": "A product created for testing",
		"price":       19.99,
		"sku":         "TEST-SKU-001",
		"status":      "active",
		"inventory":   100,
		"tags":        []string{"test", "api"},
		"customFields": map[string]interface{}{
			"testField": "test value",
		},
	}

	// Make request to create product
	resp, body := MakeRequest(t, http.MethodPost, apiPrefix+"/products", newProduct)

	// Check status code
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	// Parse response
	var createdProduct models.Product
	ParseJSONResponse(t, body, &createdProduct)

	// Verify product was created with correct data
	if createdProduct.Title != newProduct["title"] {
		t.Errorf("Expected title %s, got %s", newProduct["title"], createdProduct.Title)
	}
	if createdProduct.Price != newProduct["price"] {
		t.Errorf("Expected price %v, got %v", newProduct["price"], createdProduct.Price)
	}
	if createdProduct.SKU != newProduct["sku"] {
		t.Errorf("Expected SKU %s, got %s", newProduct["sku"], createdProduct.SKU)
	}
	if createdProduct.ID == "" {
		t.Error("Expected non-empty ID")
	}

	// Test creating a product with invalid data
	invalidProduct := map[string]interface{}{
		"price": -10, // Invalid price
	}

	resp, _ = MakeRequest(t, http.MethodPost, apiPrefix+"/products", invalidProduct)

	// Check status code (should be 400)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

// TestUpdateProduct tests the PUT /products/{id} endpoint
func TestUpdateProduct(t *testing.T) {
	// First, create a product to update
	newProduct := map[string]interface{}{
		"title":       "Product To Update",
		"description": "This product will be updated",
		"price":       29.99,
		"sku":         "UPDATE-SKU-001",
		"status":      "active",
		"inventory":   50,
	}

	resp, body := MakeRequest(t, http.MethodPost, apiPrefix+"/products", newProduct)
	var createdProduct models.Product
	ParseJSONResponse(t, body, &createdProduct)

	// Update the product
	updatedProduct := map[string]interface{}{
		"title":       "Updated Product",
		"description": "This product has been updated",
		"price":       39.99,
		"sku":         createdProduct.SKU,
		"status":      "active",
		"inventory":   75,
	}

	resp, body = MakeRequest(t, http.MethodPut, fmt.Sprintf("%s/products/%s", apiPrefix, createdProduct.ID), updatedProduct)

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Parse response
	var returnedProduct models.Product
	ParseJSONResponse(t, body, &returnedProduct)

	// Verify product was updated
	if returnedProduct.Title != updatedProduct["title"] {
		t.Errorf("Expected title %s, got %s", updatedProduct["title"], returnedProduct.Title)
	}
	if returnedProduct.Price != updatedProduct["price"] {
		t.Errorf("Expected price %v, got %v", updatedProduct["price"], returnedProduct.Price)
	}
	if returnedProduct.Inventory != updatedProduct["inventory"] {
		t.Errorf("Expected inventory %v, got %v", updatedProduct["inventory"], returnedProduct.Inventory)
	}

	// Test updating a non-existent product
	resp, _ = MakeRequest(t, http.MethodPut, fmt.Sprintf("%s/products/non-existent-id", apiPrefix), updatedProduct)

	// Check status code (should be 404)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

// TestDeleteProduct tests the DELETE /products/{id} endpoint
func TestDeleteProduct(t *testing.T) {
	// First, create a product to delete
	newProduct := map[string]interface{}{
		"title":       "Product To Delete",
		"description": "This product will be deleted",
		"price":       15.99,
		"sku":         "DELETE-SKU-001",
		"status":      "active",
		"inventory":   30,
	}

	resp, body := MakeRequest(t, http.MethodPost, apiPrefix+"/products", newProduct)
	var createdProduct models.Product
	ParseJSONResponse(t, body, &createdProduct)

	// Delete the product
	resp, _ = MakeRequest(t, http.MethodDelete, fmt.Sprintf("%s/products/%s", apiPrefix, createdProduct.ID), nil)

	// Check status code
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, resp.StatusCode)
	}

	// Verify the product was deleted by trying to get it
	resp, _ = MakeRequest(t, http.MethodGet, fmt.Sprintf("%s/products/%s", apiPrefix, createdProduct.ID), nil)

	// Check status code (should be 404)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}

	// Test deleting a non-existent product
	resp, _ = MakeRequest(t, http.MethodDelete, fmt.Sprintf("%s/products/non-existent-id", apiPrefix), nil)

	// Check status code (should be 404)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

// TestCountProducts is an integration test for the GET /products/count endpoint.
// Note: This test may require retry logic to handle DynamoDB eventual consistency.
func TestCountProducts(t *testing.T) {
	// First, get the current count
	resp, body := MakeRequest(t, http.MethodGet, apiPrefix+"/products/count", nil)

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Parse response
	var countResponse struct {
		Count int `json:"count"`
	}
	ParseJSONResponse(t, body, &countResponse)
	initialCount := countResponse.Count

	t.Logf("Initial product count: %d", initialCount)

	// Create a new product
	newProduct := map[string]interface{}{
		"title":       "Test Count Product",
		"description": "A product for testing the count endpoint",
		"price":       15.99,
		"sku":         "TEST-COUNT-001",
		"status":      "active",
		"inventory":   30,
	}

	// Make request to create product
	createResp, createBody := MakeRequest(t, http.MethodPost, apiPrefix+"/products", newProduct)
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create test product: %d", createResp.StatusCode)
	}

	// Parse the response to get the product ID
	var createdProduct models.Product
	ParseJSONResponse(t, createBody, &createdProduct)

	t.Logf("Created product with ID: %s", createdProduct.ID)

	// Retry logic for eventual consistency
	success := false
	maxRetries := 5

	for i := 0; i < maxRetries && !success; i++ {
		// Add increasing delay with each attempt
		time.Sleep(time.Duration(100+i*100) * time.Millisecond)

		// Get the count again
		resp, body = MakeRequest(t, http.MethodGet, apiPrefix+"/products/count", nil)

		// Check status code
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}

		// Parse response
		ParseJSONResponse(t, body, &countResponse)
		newCount := countResponse.Count

		t.Logf("Attempt %d: Product count after creation: %d", i+1, newCount)

		// Check if count has increased
		if newCount > initialCount {
			success = true
			break
		}
	}

	// Verify count increased
	if !success {
		t.Error("Expected product count to increase after creating a product, but it did not")
	}
}

// TestProductsFiltering tests the filtering capabilities of the GET /products endpoint
func TestProductsFiltering(t *testing.T) {
	// Create products with different statuses for testing
	activeProduct := map[string]interface{}{
		"title":       "Active Test Product",
		"description": "A product for testing filters",
		"price":       29.99,
		"sku":         "FILTER-ACTIVE-001",
		"status":      "active",
		"inventory":   50,
		"tags":        []string{"filter", "test", "active"},
	}

	draftProduct := map[string]interface{}{
		"title":       "Draft Test Product",
		"description": "A draft product for testing filters",
		"price":       19.99,
		"sku":         "FILTER-DRAFT-001",
		"status":      "draft",
		"inventory":   0,
		"tags":        []string{"filter", "test", "draft"},
	}

	// Create the products
	MakeRequest(t, http.MethodPost, apiPrefix+"/products", activeProduct)
	MakeRequest(t, http.MethodPost, apiPrefix+"/products", draftProduct)

	// Test status filter
	resp, body := MakeRequest(t, http.MethodGet, apiPrefix+"/products?status=active", nil)

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Parse response
	var activeProductsResponse struct {
		Items []models.Product `json:"items"`
	}
	ParseJSONResponse(t, body, &activeProductsResponse)

	// Verify all products have active status
	for _, product := range activeProductsResponse.Items {
		if product.Status != "active" {
			t.Errorf("Expected all products to have status 'active', found product with status '%s'", product.Status)
		}
	}

	// Test tag filter
	resp, body = MakeRequest(t, http.MethodGet, apiPrefix+"/products?tag=draft", nil)

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Parse response
	var draftTagProductsResponse struct {
		Items []models.Product `json:"items"`
	}
	ParseJSONResponse(t, body, &draftTagProductsResponse)

	// Verify products have the draft tag
	foundDraftTag := false
	for _, product := range draftTagProductsResponse.Items {
		for _, tag := range product.Tags {
			if tag == "draft" {
				foundDraftTag = true
				break
			}
		}
	}

	if !foundDraftTag && len(draftTagProductsResponse.Items) > 0 {
		t.Error("Expected to find products with 'draft' tag")
	}
}

// TestProductVariants tests the GET /products/{id}/variants endpoint
func TestProductVariants(t *testing.T) {
	// Create a product with variants for testing
	newProduct := map[string]interface{}{
		"title":       "Variant Test Product",
		"description": "A product for testing variants",
		"price":       39.99,
		"sku":         "VARIANT-TEST-001",
		"status":      "active",
		"inventory":   100,
		"variants": []map[string]interface{}{
			{
				"sku":       "VARIANT-TEST-001-S",
				"title":     "Small",
				"price":     39.99,
				"inventory": 30,
				"options": []map[string]string{
					{"name": "Size", "value": "Small"},
				},
			},
			{
				"sku":       "VARIANT-TEST-001-M",
				"title":     "Medium",
				"price":     39.99,
				"inventory": 40,
				"options": []map[string]string{
					{"name": "Size", "value": "Medium"},
				},
			},
			{
				"sku":       "VARIANT-TEST-001-L",
				"title":     "Large",
				"price":     44.99,
				"inventory": 30,
				"options": []map[string]string{
					{"name": "Size", "value": "Large"},
				},
			},
		},
	}

	// Create the product
	resp, body := MakeRequest(t, http.MethodPost, apiPrefix+"/products", newProduct)
	var createdProduct models.Product
	ParseJSONResponse(t, body, &createdProduct)

	// Test getting variants
	resp, body = MakeRequest(t, http.MethodGet, fmt.Sprintf("%s/products/%s/variants", apiPrefix, createdProduct.ID), nil)

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Parse response
	var variantsResponse struct {
		Items []models.ProductVariant `json:"items"`
		Links models.PaginationLinks  `json:"links"`
	}
	ParseJSONResponse(t, body, &variantsResponse)

	// Verify we have the correct number of variants
	expectedVariantCount := len(newProduct["variants"].([]map[string]interface{}))
	if len(variantsResponse.Items) != expectedVariantCount {
		t.Errorf("Expected %d variants, got %d", expectedVariantCount, len(variantsResponse.Items))
	}

	// Verify variant data
	for _, variant := range variantsResponse.Items {
		if variant.ProductID != createdProduct.ID {
			t.Errorf("Expected product ID %s, got %s", createdProduct.ID, variant.ProductID)
		}
		if variant.ProductTitle != createdProduct.Title {
			t.Errorf("Expected product title %s, got %s", createdProduct.Title, variant.ProductTitle)
		}
	}

	// Test getting variants for a non-existent product
	resp, _ = MakeRequest(t, http.MethodGet, fmt.Sprintf("%s/products/non-existent-id/variants", apiPrefix), nil)

	// Check status code (should be 404)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

// TestAllVariants tests the GET /products/variants endpoint
func TestAllVariants(t *testing.T) {
	// Make request to list all variants
	resp, body := MakeRequest(t, http.MethodGet, apiPrefix+"/products/variants", nil)

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Parse response
	var variantsResponse struct {
		Items []models.ProductVariant `json:"items"`
		Links models.PaginationLinks  `json:"links"`
	}
	ParseJSONResponse(t, body, &variantsResponse)

	// We just verify the response structure is correct
	// since the number of variants depends on the test execution order

	// Check that the self link is set
	if variantsResponse.Links.Self == "" {
		t.Error("Expected self link, got none")
	}
}

// TestProductSingleTableDesignAPI verifies that the product API works correctly with the single table design
func TestProductSingleTableDesignAPI(t *testing.T) {
	// Set up test environment
	handler, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create a product
	product := createTestProduct(t, handler)

	// Verify product was saved with correct key structure by checking we can retrieve it
	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/products/%s", product.ID), nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK")

	// Decode response body
	var retrievedProduct models.Product
	err := json.NewDecoder(w.Body).Decode(&retrievedProduct)
	assert.NoError(t, err, "Failed to decode response body")

	// Verify product data
	assert.Equal(t, product.ID, retrievedProduct.ID, "Product ID mismatch")
	assert.Equal(t, product.Title, retrievedProduct.Title, "Product title mismatch")

	// Test key format directly
	pk, sk := services.ProductKey(product.ID)
	expectedPK := fmt.Sprintf("%s#%s", services.EntityProduct, product.ID)
	expectedSK := fmt.Sprintf("%s#%s", services.EntityProduct, product.ID)

	assert.Equal(t, expectedPK, pk, "Product PK format doesn't match expected pattern")
	assert.Equal(t, expectedSK, sk, "Product SK format doesn't match expected pattern")
}
