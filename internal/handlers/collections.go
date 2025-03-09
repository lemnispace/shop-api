package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/internal/services"
	"github.com/lemnispace/shop-api/internal/utils"
)

// Reference to the collection service
var collectionService services.CollectionService

// SetCollectionService sets the collection service for the handlers
func SetCollectionService(service services.CollectionService) {
	collectionService = service
}

// CollectionsHandler handles requests to /v1/collections
func CollectionsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ListAllCollections(w, r)
	case http.MethodPost:
		CreateCollection(w, r)
	default:
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// CollectionDetailHandler handles requests to /v1/collections/{collectionId}
func CollectionDetailHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	apiPrefix := "/v1/collections/"
	if !strings.HasPrefix(path, apiPrefix) {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid path")
		return
	}

	// Extract collectionId from URL
	parts := strings.Split(strings.TrimPrefix(path, apiPrefix), "/")
	if len(parts) == 0 || parts[0] == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Collection ID is required")
		return
	}
	collectionID := parts[0]

	// Check if we have a products subpath
	if len(parts) > 1 && parts[1] == "products" {
		HandleCollectionProducts(w, r, collectionID, parts)
		return
	}

	switch r.Method {
	case http.MethodGet:
		GetCollection(w, r, collectionID)
	case http.MethodPut:
		UpdateCollection(w, r, collectionID)
	case http.MethodDelete:
		DeleteCollection(w, r, collectionID)
	default:
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// CollectionCountHandler handles requests to /v1/collections/count
func CollectionCountHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse query parameters for filtering
	queryParams := r.URL.Query()
	filters := buildFilterParams(queryParams)

	if collectionService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	// Get count from service
	count, err := collectionService.CountCollections(r.Context(), filters)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to count collections")
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"count": count,
	})
}

// HandleCollectionProducts handles product-related operations for a collection
func HandleCollectionProducts(w http.ResponseWriter, r *http.Request, collectionID string, pathParts []string) {
	if collectionService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	// List products in the collection
	if r.Method == http.MethodGet && len(pathParts) == 2 {
		ListCollectionProducts(w, r, collectionID)
		return
	}

	// Add a product to the collection
	if r.Method == http.MethodPost && len(pathParts) == 2 {
		AddProductToCollection(w, r, collectionID)
		return
	}

	// Remove a product from the collection
	if r.Method == http.MethodDelete && len(pathParts) == 3 {
		productID := pathParts[2]
		RemoveProductFromCollection(w, r, collectionID, productID)
		return
	}

	utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
}

// ListAllCollections lists all collections
func ListAllCollections(w http.ResponseWriter, r *http.Request) {
	utils.DebugLog("Listing all collections")

	// Parse pagination parameters
	limit, cursor := getPaginationParams(r)
	utils.DebugLog("Pagination params - limit: %d, cursor: %s", limit, cursor)

	// Parse query parameters for filtering and sorting
	queryParams := r.URL.Query()
	filters := buildFilterParams(queryParams)
	sortKey, sortOrder := getSortParams(queryParams)
	utils.DebugLog("Filter and sort params - filters: %v, sortKey: %s, sortOrder: %s", filters, sortKey, sortOrder)

	if collectionService == nil {
		utils.ErrorLog("Collection service not initialized in ListAllCollections handler")
		utils.ErrorResponse(w, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	// Get collections from service
	result, err := collectionService.ListCollections(r.Context(), limit, cursor, filters, sortKey, sortOrder)
	if err != nil {
		utils.ErrorLog("Failed to fetch collections: %v", err)
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to fetch collections")
		return
	}

	// Create self link
	selfLink := fmt.Sprintf("/v1/collections")
	if len(queryParams) > 0 {
		selfLink += "?" + queryParams.Encode()
	}

	// Create next link if there's a next cursor
	var nextLink string
	if result.NextCursor != "" {
		// Create a new query with the next cursor
		nextQueryValues := r.URL.Query()
		nextQueryValues.Set("cursor", result.NextCursor)

		nextLink = fmt.Sprintf("/v1/collections?%s", nextQueryValues.Encode())
	}

	utils.DebugLog("Found %d collections", len(result.Collections))

	// Format response according to API spec and test expectations
	response := struct {
		Items []models.Collection    `json:"items"`
		Links models.PaginationLinks `json:"links"`
	}{
		Items: result.Collections,
		Links: models.PaginationLinks{
			Self: selfLink,
			Next: nextLink,
		},
	}

	utils.DebugLog("Returning collections response with links - self: %s, next: %s",
		response.Links.Self, response.Links.Next)
	utils.JSONResponse(w, http.StatusOK, response)
}

// CreateCollection creates a new collection
func CreateCollection(w http.ResponseWriter, r *http.Request) {
	// Decode the request body
	var input models.CollectionInput
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate the collection input
	if err := validateCollectionInput(input); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if collectionService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	// Create collection instance
	collection := models.Collection{
		Title:       input.Title,
		Description: input.Description,
		Products:    []models.Product{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Create the collection
	err = collectionService.CreateCollection(r.Context(), &collection)
	if err != nil {
		log.Printf("Error creating collection: %v", err)
		utils.ErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create collection: %v", err))
		return
	}

	// Add products to the collection if provided
	for _, productID := range input.ProductIDs {
		err := collectionService.AddProductToCollection(r.Context(), collection.ID, productID)
		if err != nil {
			// Just log the error and continue, we don't want to fail the whole operation
			// In a real implementation, you might want better error handling
			continue
		}
	}

	// Get the updated collection with products
	updatedCollection, err := collectionService.GetCollection(r.Context(), collection.ID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve created collection")
		return
	}

	utils.JSONResponse(w, http.StatusCreated, updatedCollection)
}

// GetCollection gets a collection by ID
func GetCollection(w http.ResponseWriter, r *http.Request, collectionID string) {
	if collectionService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	// Get collection from service
	collection, err := collectionService.GetCollection(r.Context(), collectionID)
	if err != nil {
		if err == services.ErrCollectionNotFound {
			utils.ErrorResponse(w, http.StatusNotFound, "Collection not found")
		} else {
			utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to fetch collection")
		}
		return
	}

	utils.JSONResponse(w, http.StatusOK, collection)
}

// UpdateCollection updates a collection
func UpdateCollection(w http.ResponseWriter, r *http.Request, collectionID string) {
	// Decode the request body
	var input models.CollectionInput
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate the collection input
	if err := validateCollectionInput(input); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if collectionService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	// Get the existing collection
	existingCollection, err := collectionService.GetCollection(r.Context(), collectionID)
	if err != nil {
		if err == services.ErrCollectionNotFound {
			utils.ErrorResponse(w, http.StatusNotFound, "Collection not found")
		} else {
			utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to fetch collection")
		}
		return
	}

	// Update the collection fields
	existingCollection.Title = input.Title
	existingCollection.Description = input.Description
	existingCollection.UpdatedAt = time.Now()

	// Save the updated collection
	err = collectionService.UpdateCollection(r.Context(), existingCollection)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to update collection")
		return
	}

	// Clear products and add the new ones if provided
	// In a real implementation, you might want to be more selective about this
	// (e.g., only remove products that are no longer in the list)
	for _, product := range existingCollection.Products {
		_ = collectionService.RemoveProductFromCollection(r.Context(), collectionID, product.ID)
	}

	for _, productID := range input.ProductIDs {
		_ = collectionService.AddProductToCollection(r.Context(), collectionID, productID)
	}

	// Get the updated collection with products
	updatedCollection, err := collectionService.GetCollection(r.Context(), collectionID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve updated collection")
		return
	}

	utils.JSONResponse(w, http.StatusOK, updatedCollection)
}

// DeleteCollection deletes a collection
func DeleteCollection(w http.ResponseWriter, r *http.Request, collectionID string) {
	utils.DebugLog("Deleting collection with ID: %s", collectionID)

	if collectionService == nil {
		utils.ErrorLog("Collection service not initialized in DeleteCollection handler")
		utils.ErrorResponse(w, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	// Delete the collection
	err := collectionService.DeleteCollection(r.Context(), collectionID)
	if err != nil {
		if err == services.ErrCollectionNotFound {
			utils.DebugLog("Collection not found for deletion: %s", collectionID)
			utils.ErrorResponse(w, http.StatusNotFound, "Collection not found")
		} else {
			utils.ErrorLog("Failed to delete collection %s: %v", collectionID, err)
			utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to delete collection")
		}
		return
	}

	// Return 204 No Content on successful deletion
	utils.DebugLog("Successfully deleted collection: %s", collectionID)
	w.WriteHeader(http.StatusNoContent)
}

// ListCollectionProducts lists the products in a collection
func ListCollectionProducts(w http.ResponseWriter, r *http.Request, collectionID string) {
	utils.DebugLog("Listing products for collection: %s", collectionID)
	
	// Parse pagination parameters
	limit, cursor := getPaginationParams(r)
	utils.DebugLog("Pagination params - limit: %d, cursor: %s", limit, cursor)

	if collectionService == nil {
		utils.ErrorLog("Collection service not initialized in ListCollectionProducts handler")
		utils.ErrorResponse(w, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	// Get products from service
	products, nextCursor, err := collectionService.ListCollectionProducts(r.Context(), collectionID, limit, cursor)
	if err != nil {
		if err == services.ErrCollectionNotFound {
			utils.ErrorLog("Collection not found: %s", collectionID)
			utils.ErrorResponse(w, http.StatusNotFound, "Collection not found")
		} else {
			utils.ErrorLog("Failed to fetch collection products: %v", err)
			utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to fetch collection products")
		}
		return
	}

	utils.DebugLog("Found %d products in collection %s", len(products), collectionID)
	
	// Log product IDs for debugging
	for i, product := range products {
		utils.DebugLog("Product %d: ID=%s, Title=%s", i, product.ID, product.Title)
	}

	// Create self link
	selfLink := fmt.Sprintf("/v1/collections/%s/products", collectionID)
	if r.URL.RawQuery != "" {
		selfLink += "?" + r.URL.RawQuery
	}
	
	// Create next link if there's a next cursor
	var nextLink string
	if nextCursor != "" {
		nextQueryValues := r.URL.Query()
		nextQueryValues.Set("cursor", nextCursor)
		
		nextLink = fmt.Sprintf("/v1/collections/%s/products?%s", collectionID, nextQueryValues.Encode())
	}

	// Format response according to API spec and test expectations
	response := struct {
		Items []models.Product       `json:"items"`
		Links models.PaginationLinks `json:"links"`
	}{
		Items: products,
		Links: models.PaginationLinks{
			Self: selfLink,
			Next: nextLink,
		},
	}

	utils.DebugLog("Returning products response with links - self: %s, next: %s", 
		response.Links.Self, response.Links.Next)
	utils.JSONResponse(w, http.StatusOK, response)
}

// AddProductToCollection adds a product to a collection
func AddProductToCollection(w http.ResponseWriter, r *http.Request, collectionID string) {
	utils.DebugLog("Adding product to collection: %s", collectionID)
	
	// Decode the request body
	var input struct {
		ProductID  string   `json:"productId"`
		ProductIDs []string `json:"productIds"`
	}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		utils.ErrorLog("Invalid request body: %v", err)
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if collectionService == nil {
		utils.ErrorLog("Collection service not initialized in AddProductToCollection handler")
		utils.ErrorResponse(w, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	// Verify collection exists
	_, err = collectionService.GetCollection(r.Context(), collectionID)
	if err != nil {
		if err == services.ErrCollectionNotFound {
			utils.ErrorLog("Collection not found: %s", collectionID)
			utils.ErrorResponse(w, http.StatusNotFound, "Collection not found")
		} else {
			utils.ErrorLog("Failed to fetch collection: %v", err)
			utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to fetch collection")
		}
		return
	}

	// Process single productId or multiple productIds
	productsToAdd := []string{}

	if input.ProductID != "" {
		productsToAdd = append(productsToAdd, input.ProductID)
	}

	if len(input.ProductIDs) > 0 {
		productsToAdd = append(productsToAdd, input.ProductIDs...)
	}

	if len(productsToAdd) == 0 {
		utils.ErrorLog("No product IDs provided")
		utils.ErrorResponse(w, http.StatusBadRequest, "At least one product ID is required")
		return
	}

	utils.DebugLog("Adding %d products to collection %s", len(productsToAdd), collectionID)

	// Add each product to the collection
	for _, productID := range productsToAdd {
		utils.DebugLog("Adding product %s to collection %s", productID, collectionID)
		err = collectionService.AddProductToCollection(r.Context(), collectionID, productID)
		if err != nil {
			if err == services.ErrCollectionNotFound {
				utils.ErrorLog("Collection not found: %s", collectionID)
				utils.ErrorResponse(w, http.StatusNotFound, "Collection not found")
				return
			} else if err == services.ErrProductNotFound {
				utils.ErrorLog("Product not found: %s", productID)
				// Log the error but continue with other products
				continue
			} else {
				utils.ErrorLog("Failed to add product to collection: %v", err)
				utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to add product to collection")
				return
			}
		}
	}

	// Return 204 No Content on successful addition of products
	utils.DebugLog("Successfully added products to collection %s", collectionID)
	w.WriteHeader(http.StatusNoContent)
}

// RemoveProductFromCollection removes a product from a collection
func RemoveProductFromCollection(w http.ResponseWriter, r *http.Request, collectionID, productID string) {
	if collectionService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	// Remove the product from the collection
	err := collectionService.RemoveProductFromCollection(r.Context(), collectionID, productID)
	if err != nil {
		if err == services.ErrCollectionNotFound {
			utils.ErrorResponse(w, http.StatusNotFound, "Collection not found")
		} else {
			utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to remove product from collection")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// validateCollectionInput validates the collection input data
func validateCollectionInput(input models.CollectionInput) error {
	if input.Title == "" {
		return &validationError{Field: "title", Message: "Title is required"}
	}
	return nil
}
