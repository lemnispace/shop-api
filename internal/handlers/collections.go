package handlers

import (
	"encoding/json"
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
	// Parse pagination parameters
	limit, cursor := getPaginationParams(r)

	// Parse query parameters for filtering and sorting
	queryParams := r.URL.Query()
	filters := buildFilterParams(queryParams)
	sortKey, sortOrder := getSortParams(queryParams)

	if collectionService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	// Initialize response structure
	response := models.PaginatedResponse{
		Items: []interface{}{},
		Links: models.PaginationLinks{
			Self: r.URL.String(),
		},
	}

	// Get collections from service
	result, err := collectionService.ListCollections(r.Context(), limit, cursor, filters, sortKey, sortOrder)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to fetch collections")
		return
	}

	// Populate response with collections
	for _, collection := range result.Collections {
		response.Items = append(response.Items, collection)
	}

	// Add pagination links
	if result.NextCursor != "" {
		nextURL := buildNextPageURL(r.URL, result.NextCursor)
		response.Links.Next = nextURL
	}

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
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to create collection")
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
	if collectionService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	// Delete the collection
	err := collectionService.DeleteCollection(r.Context(), collectionID)
	if err != nil {
		if err == services.ErrCollectionNotFound {
			utils.ErrorResponse(w, http.StatusNotFound, "Collection not found")
		} else {
			utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to delete collection")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListCollectionProducts lists the products in a collection
func ListCollectionProducts(w http.ResponseWriter, r *http.Request, collectionID string) {
	// Parse pagination parameters
	limit, cursor := getPaginationParams(r)

	if collectionService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	// Get products from service
	products, nextCursor, err := collectionService.ListCollectionProducts(r.Context(), collectionID, limit, cursor)
	if err != nil {
		if err == services.ErrCollectionNotFound {
			utils.ErrorResponse(w, http.StatusNotFound, "Collection not found")
		} else {
			utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to fetch collection products")
		}
		return
	}

	// Initialize response with pagination
	response := models.PaginatedResponse{
		Items: make([]interface{}, len(products)),
		Links: models.PaginationLinks{
			Self: r.URL.String(),
		},
	}

	// Populate items
	for i, product := range products {
		response.Items[i] = product
	}

	// Add pagination links
	if nextCursor != "" {
		nextURL := buildNextPageURL(r.URL, nextCursor)
		response.Links.Next = nextURL
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// AddProductToCollection adds a product to a collection
func AddProductToCollection(w http.ResponseWriter, r *http.Request, collectionID string) {
	// Decode the request body
	var input struct {
		ProductID string `json:"productId"`
	}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if input.ProductID == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Product ID is required")
		return
	}

	if collectionService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	// Add the product to the collection
	err = collectionService.AddProductToCollection(r.Context(), collectionID, input.ProductID)
	if err != nil {
		if err == services.ErrCollectionNotFound {
			utils.ErrorResponse(w, http.StatusNotFound, "Collection not found")
		} else if err == services.ErrProductNotFound {
			utils.ErrorResponse(w, http.StatusNotFound, "Product not found")
		} else {
			utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to add product to collection")
		}
		return
	}

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
