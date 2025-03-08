package api

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/lemnispace/shop-api/internal/models"
)

// TestGetProducts tests the GET /products endpoint
func TestGetProducts(t *testing.T) {
	// Make request to list all products
	resp, body := MakeRequest(t, http.MethodGet, apiPrefix+"/products", nil)

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Parse response
	var response struct {
		Items []models.Product       `json:"items"`
		Links models.PaginationLinks `json:"links"`
	}
	ParseJSONResponse(t, body, &response)

	// Verify we have some products
	if len(response.Items) == 0 {
		t.Error("Expected products, got none")
	}

	// Check that the self link is set
	if response.Links.Self == "" {
		t.Error("Expected self link, got none")
	}
}

// TestGetProductByID tests the GET /products/{id} endpoint
func TestGetProductByID(t *testing.T) {
	// First, get all products to find a valid ID
	resp, body := MakeRequest(t, http.MethodGet, apiPrefix+"/products", nil)
	var productsResponse struct {
		Items []models.Product `json:"items"`
	}
	ParseJSONResponse(t, body, &productsResponse)

	if len(productsResponse.Items) == 0 {
		t.Fatal("No products available for testing")
	}

	// Use the ID of the first product
	productID := productsResponse.Items[0].ID

	// Test getting a product by ID
	resp, body = MakeRequest(t, http.MethodGet, fmt.Sprintf("%s/products/%s", apiPrefix, productID), nil)

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Parse response
	var product models.Product
	ParseJSONResponse(t, body, &product)

	// Verify we got the right product
	if product.ID != productID {
		t.Errorf("Expected product ID %s, got %s", productID, product.ID)
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

// TestCountProducts tests the GET /products/count endpoint
func TestCountProducts(t *testing.T) {
	// Make request to count products
	resp, body := MakeRequest(t, http.MethodGet, apiPrefix+"/products/count", nil)

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Parse response
	var countResponse struct {
		Count int `json:"count"`
	}
	ParseJSONResponse(t, body, &countResponse)

	// Verify count is non-negative
	if countResponse.Count < 0 {
		t.Errorf("Expected non-negative count, got %d", countResponse.Count)
	}

	// Create a new product to check if count increases
	initialCount := countResponse.Count

	newProduct := map[string]interface{}{
		"title":       "Count Test Product",
		"description": "A product for testing count",
		"price":       9.99,
		"sku":         "COUNT-SKU-001",
		"status":      "active",
		"inventory":   10,
	}

	MakeRequest(t, http.MethodPost, apiPrefix+"/products", newProduct)

	// Check count again
	resp, body = MakeRequest(t, http.MethodGet, apiPrefix+"/products/count", nil)
	ParseJSONResponse(t, body, &countResponse)

	// Verify count increased by 1
	if countResponse.Count != initialCount+1 {
		t.Errorf("Expected count to increase by 1 from %d to %d, got %d", initialCount, initialCount+1, countResponse.Count)
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
