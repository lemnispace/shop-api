package api

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/lemnispace/shop-api/internal/models"
)

// TestGetCollections tests the GET /collections endpoint
func TestGetCollections(t *testing.T) {
	// Make request to list all collections
	resp, body := MakeRequest(t, http.MethodGet, apiPrefix+"/collections", nil)

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Parse response
	var response struct {
		Items []models.Collection    `json:"items"`
		Links models.PaginationLinks `json:"links"`
	}
	ParseJSONResponse(t, body, &response)

	// Verify we have some collections
	if len(response.Items) == 0 {
		t.Error("Expected collections, got none")
	}

	// Check that the self link is set
	if response.Links.Self == "" {
		t.Error("Expected self link, got none")
	}
}

// TestGetCollectionByID tests the GET /collections/{id} endpoint
func TestGetCollectionByID(t *testing.T) {
	// First, get all collections to find a valid ID
	resp, body := MakeRequest(t, http.MethodGet, apiPrefix+"/collections", nil)
	var collectionsResponse struct {
		Items []models.Collection `json:"items"`
	}
	ParseJSONResponse(t, body, &collectionsResponse)

	if len(collectionsResponse.Items) == 0 {
		t.Fatal("No collections available for testing")
	}

	// Use the ID of the first collection
	collectionID := collectionsResponse.Items[0].ID

	// Test getting a collection by ID
	resp, body = MakeRequest(t, http.MethodGet, fmt.Sprintf("%s/collections/%s", apiPrefix, collectionID), nil)

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Parse response
	var collection models.Collection
	ParseJSONResponse(t, body, &collection)

	// Verify we got the right collection
	if collection.ID != collectionID {
		t.Errorf("Expected collection ID %s, got %s", collectionID, collection.ID)
	}

	// Test getting a non-existent collection
	resp, _ = MakeRequest(t, http.MethodGet, fmt.Sprintf("%s/collections/non-existent-id", apiPrefix), nil)

	// Check status code (should be 404)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

// TestCreateCollection tests the POST /collections endpoint
func TestCreateCollection(t *testing.T) {
	// Create a new collection
	newCollection := map[string]interface{}{
		"title":       "Test Collection",
		"description": "A collection created for testing",
		"productIds":  []string{},
	}

	// Make request to create collection
	resp, body := MakeRequest(t, http.MethodPost, apiPrefix+"/collections", newCollection)

	// Check status code
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	// Parse response
	var createdCollection models.Collection
	ParseJSONResponse(t, body, &createdCollection)

	// Verify collection was created with correct data
	if createdCollection.Title != newCollection["title"] {
		t.Errorf("Expected title %s, got %s", newCollection["title"], createdCollection.Title)
	}
	if createdCollection.Description != newCollection["description"] {
		t.Errorf("Expected description %s, got %s", newCollection["description"], createdCollection.Description)
	}
	if createdCollection.ID == "" {
		t.Error("Expected non-empty ID")
	}

	// Test creating a collection with invalid data (no title)
	invalidCollection := map[string]interface{}{
		"description": "Invalid collection without title",
	}

	resp, _ = MakeRequest(t, http.MethodPost, apiPrefix+"/collections", invalidCollection)

	// Check status code (should be 400)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

// TestUpdateCollection tests the PUT /collections/{id} endpoint
func TestUpdateCollection(t *testing.T) {
	// First, create a collection to update
	newCollection := map[string]interface{}{
		"title":       "Collection To Update",
		"description": "This collection will be updated",
		"productIds":  []string{},
	}

	resp, body := MakeRequest(t, http.MethodPost, apiPrefix+"/collections", newCollection)
	var createdCollection models.Collection
	ParseJSONResponse(t, body, &createdCollection)

	// Update the collection
	updatedCollection := map[string]interface{}{
		"title":       "Updated Collection",
		"description": "This collection has been updated",
		"productIds":  []string{},
	}

	resp, body = MakeRequest(t, http.MethodPut, fmt.Sprintf("%s/collections/%s", apiPrefix, createdCollection.ID), updatedCollection)

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Parse response
	var returnedCollection models.Collection
	ParseJSONResponse(t, body, &returnedCollection)

	// Verify collection was updated
	if returnedCollection.Title != updatedCollection["title"] {
		t.Errorf("Expected title %s, got %s", updatedCollection["title"], returnedCollection.Title)
	}
	if returnedCollection.Description != updatedCollection["description"] {
		t.Errorf("Expected description %s, got %s", updatedCollection["description"], returnedCollection.Description)
	}

	// Test updating a non-existent collection
	resp, _ = MakeRequest(t, http.MethodPut, fmt.Sprintf("%s/collections/non-existent-id", apiPrefix), updatedCollection)

	// Check status code (should be 404)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

// TestDeleteCollection tests the DELETE /collections/{id} endpoint
func TestDeleteCollection(t *testing.T) {
	// First, create a collection to delete
	newCollection := map[string]interface{}{
		"title":       "Collection To Delete",
		"description": "This collection will be deleted",
		"productIds":  []string{},
	}

	resp, body := MakeRequest(t, http.MethodPost, apiPrefix+"/collections", newCollection)
	var createdCollection models.Collection
	ParseJSONResponse(t, body, &createdCollection)

	// Delete the collection
	resp, _ = MakeRequest(t, http.MethodDelete, fmt.Sprintf("%s/collections/%s", apiPrefix, createdCollection.ID), nil)

	// Check status code
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, resp.StatusCode)
	}

	// Verify the collection was deleted by trying to get it
	resp, _ = MakeRequest(t, http.MethodGet, fmt.Sprintf("%s/collections/%s", apiPrefix, createdCollection.ID), nil)

	// Check status code (should be 404)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}

	// Test deleting a non-existent collection
	resp, _ = MakeRequest(t, http.MethodDelete, fmt.Sprintf("%s/collections/non-existent-id", apiPrefix), nil)

	// Check status code (should be 404)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

// TestCountCollections tests the GET /collections/count endpoint
func TestCountCollections(t *testing.T) {
	// Make request to count collections
	resp, body := MakeRequest(t, http.MethodGet, apiPrefix+"/collections/count", nil)

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

	// Create a new collection to check if count increases
	initialCount := countResponse.Count

	newCollection := map[string]interface{}{
		"title":       "Count Test Collection",
		"description": "A collection for testing count",
		"productIds":  []string{},
	}

	MakeRequest(t, http.MethodPost, apiPrefix+"/collections", newCollection)

	// Check count again
	resp, body = MakeRequest(t, http.MethodGet, apiPrefix+"/collections/count", nil)
	ParseJSONResponse(t, body, &countResponse)

	// Verify count increased by 1
	if countResponse.Count != initialCount+1 {
		t.Errorf("Expected count to increase by 1 from %d to %d, got %d", initialCount, initialCount+1, countResponse.Count)
	}
}

// TestCollectionProducts tests the collection's product management endpoints
func TestCollectionProducts(t *testing.T) {
	// Use a unique identifier for test resources to avoid conflicts
	testID := fmt.Sprintf("test-%d", time.Now().UnixNano())
	
	// Create a new product for testing with a simple structure
	newProduct := map[string]interface{}{
		"title":       fmt.Sprintf("Collection Test Product %s", testID),
		"description": "A product for testing collections",
		"price":       25.99,
		"sku":         fmt.Sprintf("COLL-TEST-%s", testID),
		"status":      "active",
		"inventory":   10,
	}

	t.Logf("Creating test product with SKU: %s", newProduct["sku"])
	resp, body := MakeRequest(t, http.MethodPost, apiPrefix+"/products", newProduct)
	
	// Check status code
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, resp.StatusCode)
		t.Logf("Response body: %s", body)
		t.Skip("Failed to create test product, skipping rest of test")
	}
	
	var createdProduct struct {
		ID string `json:"id"`
	}
	ParseJSONResponse(t, body, &createdProduct)
	
	productID := createdProduct.ID
	if productID == "" {
		t.Fatal("Product ID is empty, cannot continue test")
	}
	
	t.Logf("Created test product with ID: %s", productID)

	// Create a new collection for testing with minimal fields
	newCollection := map[string]interface{}{
		"title":       fmt.Sprintf("Product Collection Test %s", testID),
		"description": "A collection for testing product management",
	}

	t.Logf("Creating test collection")
	resp, body = MakeRequest(t, http.MethodPost, apiPrefix+"/collections", newCollection)
	
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, resp.StatusCode)
		t.Logf("Response body: %s", body)
		t.Skip("Failed to create test collection, skipping rest of test")
	}
	
	var collection struct {
		ID string `json:"id"`
	}
	ParseJSONResponse(t, body, &collection)
	
	collectionID := collection.ID
	if collectionID == "" {
		t.Fatal("Collection ID is empty, cannot continue test")
	}
	
	t.Logf("Created test collection with ID: %s", collectionID)

	// Add a small delay before making the next request
	time.Sleep(200 * time.Millisecond)

	// Test adding a product to the collection with simple payload
	addProductBody := map[string]interface{}{
		"productId": productID,
	}

	t.Logf("Adding product %s to collection %s", productID, collectionID)
	
	// Make the request with a simple structure
	addProductURL := fmt.Sprintf("%s/collections/%s/products", apiPrefix, collectionID)
	resp, body = MakeRequest(t, http.MethodPost, addProductURL, addProductBody)

	// Check status code
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, resp.StatusCode)
		t.Logf("Response body: %s", body)
		t.Skip("Failed to add product to collection, skipping rest of test")
	}
	
	// Add a small delay before querying for the products
	time.Sleep(200 * time.Millisecond)

	// Test listing products in the collection
	t.Logf("Listing products in collection %s", collectionID)
	productsURL := fmt.Sprintf("%s/collections/%s/products", apiPrefix, collectionID)
	resp, body = MakeRequest(t, http.MethodGet, productsURL, nil)

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		t.Logf("Response body: %s", body)
		// Skip further assertions if we couldn't get the product list
		t.Skip("Failed to list products in collection")
	}

	// Parse response to check for products
	var productsListResponse struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	ParseJSONResponse(t, body, &productsListResponse)
	
	// Validate response structure
	t.Logf("Found %d products in collection response", len(productsListResponse.Items))
	for i, p := range productsListResponse.Items {
		t.Logf("Product %d: ID=%s", i, p.ID)
	}
	
	// Simple check for the product we added
	foundProduct := false
	for _, p := range productsListResponse.Items {
		if p.ID == productID {
			foundProduct = true
			break
		}
	}

	if !foundProduct {
		t.Errorf("Expected to find product %s in collection, but it was not found", productID)
	} else {
		t.Logf("Successfully found product %s in collection", productID)
	}
}
