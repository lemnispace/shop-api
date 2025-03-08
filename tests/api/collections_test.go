package api

import (
	"fmt"
	"net/http"
	"testing"

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
	// First, get all products to find a valid product ID
	resp, body := MakeRequest(t, http.MethodGet, apiPrefix+"/products", nil)
	var productsResponse struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	ParseJSONResponse(t, body, &productsResponse)

	if len(productsResponse.Items) == 0 {
		t.Skip("No products available for testing collection products")
	}

	productID := productsResponse.Items[0].ID

	// Create a new collection for testing
	newCollection := map[string]interface{}{
		"title":       "Product Collection Test",
		"description": "A collection for testing product management",
		"productIds":  []string{},
	}

	resp, body = MakeRequest(t, http.MethodPost, apiPrefix+"/collections", newCollection)
	var collection models.Collection
	ParseJSONResponse(t, body, &collection)

	// Test adding a product to the collection
	addProductBody := map[string]interface{}{
		"productId": productID,
	}

	resp, _ = MakeRequest(t, http.MethodPost, fmt.Sprintf("%s/collections/%s/products", apiPrefix, collection.ID), addProductBody)

	// Check status code
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, resp.StatusCode)
	}

	// Test listing products in the collection
	resp, body = MakeRequest(t, http.MethodGet, fmt.Sprintf("%s/collections/%s/products", apiPrefix, collection.ID), nil)

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Parse response
	var productsListResponse struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	ParseJSONResponse(t, body, &productsListResponse)

	// Verify the product was added
	foundProduct := false
	for _, p := range productsListResponse.Items {
		if p.ID == productID {
			foundProduct = true
			break
		}
	}

	if !foundProduct {
		t.Errorf("Expected to find product %s in collection, but it was not found", productID)
	}

	// Test removing a product from the collection
	resp, _ = MakeRequest(t, http.MethodDelete, fmt.Sprintf("%s/collections/%s/products/%s", apiPrefix, collection.ID, productID), nil)

	// Check status code
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, resp.StatusCode)
	}

	// Verify the product was removed
	resp, body = MakeRequest(t, http.MethodGet, fmt.Sprintf("%s/collections/%s/products", apiPrefix, collection.ID), nil)
	ParseJSONResponse(t, body, &productsListResponse)

	foundProduct = false
	for _, p := range productsListResponse.Items {
		if p.ID == productID {
			foundProduct = true
			break
		}
	}

	if foundProduct {
		t.Errorf("Expected product %s to be removed from collection, but it was still found", productID)
	}
}
