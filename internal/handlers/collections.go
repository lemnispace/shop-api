package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
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

// ListAllCollections handles GET /v1/collections
func ListAllCollections(c *gin.Context) {
	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	cursor := c.Query("cursor")

	if collectionService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	// Get collections from service
	result, err := collectionService.ListCollections(c.Request.Context(), limit, cursor, nil, "", "")
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch collections: "+err.Error())
		return
	}

	// Construct response according to API_DESIGN spec
	collectionsResponse := make([]gin.H, len(result.Collections))
	for i, coll := range result.Collections {
		// Get product count efficiently for each collection
		productCount := 0
		if count, err := collectionService.CountCollectionProducts(c.Request.Context(), coll.ID); err == nil {
			productCount = count
		}
		collectionsResponse[i] = gin.H{
			"id":           coll.ID,
			"title":        coll.Title,
			"description":  coll.Description,
			"productCount": productCount,
			"createdAt":    coll.CreatedAt,
			"updatedAt":    coll.UpdatedAt,
		}
	}

	response := gin.H{
		"collections": collectionsResponse,
		"pagination": gin.H{
			"nextCursor": result.NextCursor,
			"hasMore":    result.NextCursor != "",
		},
	}

	utils.JSONResponse(c, http.StatusOK, response)
}

// CreateCollection handles POST /v1/collections
func CreateCollection(c *gin.Context) {
	var input models.CollectionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if input.Title == "" {
		utils.ValidationError(c, "title", "Title is required")
		return
	}

	if collectionService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	collection := models.Collection{
		Title:       input.Title,
		Description: input.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := collectionService.CreateCollection(c.Request.Context(), &collection)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create collection: "+err.Error())
		return
	}

	// Add products if specified
	productCount := 0
	if len(input.ProductIDs) > 0 {
		for _, productID := range input.ProductIDs {
			err = collectionService.AddProductToCollection(c.Request.Context(), collection.ID, productID)
			if err != nil {
				fmt.Printf("Warning: Failed to add product %s during collection creation: %v\n", productID, err)
			} else {
				productCount++
			}
		}
	}

	// Return response as per API_DESIGN spec
	response := gin.H{
		"id":           collection.ID,
		"title":        collection.Title,
		"description":  collection.Description,
		"productCount": productCount, // Count based on successful additions
		"createdAt":    collection.CreatedAt,
		"updatedAt":    collection.UpdatedAt,
	}

	utils.JSONResponse(c, http.StatusCreated, response)
}

// GetCollection handles GET /v1/collections/:collectionId
func GetCollection(c *gin.Context) {
	collectionId := c.Param("collectionId")

	includeProductsStr := c.DefaultQuery("includeProducts", "true")
	includeProducts := includeProductsStr == "true"
	productLimit, _ := strconv.Atoi(c.DefaultQuery("productLimit", "20"))
	// productCursor := c.Query("productCursor") // Not used until service supports it

	if collectionService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	collection, err := collectionService.GetCollection(c.Request.Context(), collectionId)
	if err != nil {
		if err == services.ErrCollectionNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "Collection not found")
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch collection: "+err.Error())
		}
		return
	}

	// Prepare response base
	response := gin.H{
		"id":          collection.ID,
		"title":       collection.Title,
		"description": collection.Description,
		"createdAt":   collection.CreatedAt,
		"updatedAt":   collection.UpdatedAt,
	}

	if includeProducts {
		allProducts := collection.Products
		endIndex := productLimit
		if endIndex > len(allProducts) {
			endIndex = len(allProducts)
		}
		paginatedProducts := allProducts[:endIndex]

		hasMore := len(allProducts) > productLimit
		nextCursorVal := ""
		if hasMore {
			nextCursorVal = fmt.Sprintf("offset_%d", productLimit)
		}

		response["products"] = paginatedProducts
		response["productPagination"] = gin.H{
			"nextCursor": nextCursorVal,
			"hasMore":    hasMore,
		}
	} else {
		// NOTE: Cannot efficiently get product count here without another call or service change.
		// Returning length of products if available, otherwise omitting or returning placeholder.
		if collection.Products != nil {
			response["productCount"] = len(collection.Products)
		} else {
			response["productCount"] = 0 // Or omit
		}
	}

	utils.JSONResponse(c, http.StatusOK, response)
}

// UpdateCollection handles PUT /v1/collections/:collectionId
func UpdateCollection(c *gin.Context) {
	collectionId := c.Param("collectionId")

	var input models.CollectionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if input.Title == "" {
		utils.ValidationError(c, "title", "Title is required")
		return
	}

	if collectionService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	collectionToUpdate := models.Collection{
		ID:          collectionId,
		Title:       input.Title,
		Description: input.Description,
		UpdatedAt:   time.Now(),
	}

	err := collectionService.UpdateCollection(c.Request.Context(), &collectionToUpdate)
	if err != nil {
		if err == services.ErrCollectionNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "Collection not found")
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update collection: "+err.Error())
		}
		return
	}

	productCount := 0 // Placeholder count
	if input.ProductIDs != nil {
		currentCollection, err := collectionService.GetCollection(c.Request.Context(), collectionId)
		if err != nil {
			fmt.Printf("Warning: Failed to get current products for replacement: %v\n", err)
		} else {
			for _, prod := range currentCollection.Products {
				err = collectionService.RemoveProductFromCollection(c.Request.Context(), collectionId, prod.ID)
				if err != nil {
					fmt.Printf("Warning: Failed to remove product %s during replacement: %v\n", prod.ID, err)
				}
			}
		}
		for _, productID := range input.ProductIDs {
			err = collectionService.AddProductToCollection(c.Request.Context(), collectionId, productID)
			if err != nil {
				fmt.Printf("Warning: Failed to add product %s during replacement: %v\n", productID, err)
			} else {
				productCount++ // Count successfully added products
			}
		}
	} else {
		// If ProductIDs field was omitted, we need the count from the existing collection
		currentCollection, err := collectionService.GetCollection(c.Request.Context(), collectionId)
		if err == nil {
			productCount = len(currentCollection.Products)
		}
	}

	// Fetch final updated collection state to return
	finalCollection, err := collectionService.GetCollection(c.Request.Context(), collectionId)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch updated collection: "+err.Error())
		return
	}

	// Construct response based on GetCollection format (API requires this)
	response := gin.H{
		"id":           finalCollection.ID,
		"title":        finalCollection.Title,
		"description":  finalCollection.Description,
		"productCount": productCount, // Use count derived from update process
		"createdAt":    finalCollection.CreatedAt,
		"updatedAt":    finalCollection.UpdatedAt,
	}

	utils.JSONResponse(c, http.StatusOK, response)
}

// DeleteCollection handles DELETE /v1/collections/:collectionId
func DeleteCollection(c *gin.Context) {
	collectionId := c.Param("collectionId")

	if collectionService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	err := collectionService.DeleteCollection(c.Request.Context(), collectionId)
	if err != nil {
		if err == services.ErrCollectionNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "Collection not found")
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete collection: "+err.Error())
		}
		return
	}

	utils.NoContent(c)
}

// CollectionCount handles GET /v1/collections/count
func CollectionCount(c *gin.Context) {
	filters := make(map[string]interface{}) // Placeholder for potential filters

	if collectionService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	count, err := collectionService.CountCollections(c.Request.Context(), filters)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to count collections: "+err.Error())
		return
	}

	utils.JSONResponse(c, http.StatusOK, gin.H{"count": count})
}

// ListCollectionProducts handles GET /v1/collections/:collectionId/products
func ListCollectionProducts(c *gin.Context) {
	collectionId := c.Param("collectionId")
	productLimit, _ := strconv.Atoi(c.DefaultQuery("productLimit", "20"))
	// productCursor := c.Query("productCursor") // Not used until service supports it

	if collectionService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	collection, err := collectionService.GetCollection(c.Request.Context(), collectionId)
	if err != nil {
		if err == services.ErrCollectionNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "Collection not found")
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to list collection products: "+err.Error())
		}
		return
	}

	// Manual pagination based on limit (needs service improvement for cursor)
	allProducts := collection.Products
	endIndex := productLimit
	if endIndex > len(allProducts) {
		endIndex = len(allProducts)
	}
	paginatedProducts := allProducts[:endIndex]
	hasMore := len(allProducts) > productLimit
	nextCursorVal := ""
	if hasMore {
		nextCursorVal = fmt.Sprintf("offset_%d", productLimit)
	}

	// Format response
	response := gin.H{
		"products": paginatedProducts,
		"pagination": gin.H{
			"nextCursor": nextCursorVal,
			"hasMore":    hasMore,
		},
	}

	utils.JSONResponse(c, http.StatusOK, response)
}

// AddProductToCollection handles POST /v1/collections/:collectionId/products
func AddProductToCollection(c *gin.Context) {
	collectionId := c.Param("collectionId")

	var input struct {
		ProductIDs []string `json:"productIds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	if len(input.ProductIDs) == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "productIds cannot be empty")
		return
	}

	if collectionService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	// Add products individually
	addedCount := 0
	var lastErr error
	for _, productID := range input.ProductIDs {
		err := collectionService.AddProductToCollection(c.Request.Context(), collectionId, productID)
		if err != nil {
			lastErr = err // Store last error
			fmt.Printf("Warning: Failed to add product %s to collection %s: %v\n", productID, collectionId, err)
		} else {
			addedCount++
		}
	}

	// Handle potential errors
	if lastErr != nil && addedCount == 0 { // If no products were added successfully
		if lastErr == services.ErrCollectionNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "Collection not found")
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to add any products to collection: "+lastErr.Error())
		}
		return
	}

	// NOTE: Cannot efficiently get final product count without another DB call.
	// Returning count based on successful additions for now.
	finalProductCount := addedCount
	// Ideally, fetch the count after additions: finalProductCount, _ := collectionService.CountCollectionProducts(...)

	// Return response as per API_DESIGN spec
	response := gin.H{
		"success":      true,
		"collectionId": collectionId,
		"productCount": finalProductCount,
	}

	utils.JSONResponse(c, http.StatusOK, response)
}

// RemoveProductFromCollection handles DELETE /v1/collections/:collectionId/products
func RemoveProductFromCollection(c *gin.Context) {
	collectionId := c.Param("collectionId")

	var input struct {
		ProductIDs []string `json:"productIds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	if len(input.ProductIDs) == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "productIds cannot be empty")
		return
	}

	if collectionService == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Collection service not initialized")
		return
	}

	// Remove products individually
	removedCount := 0
	var lastErr error
	for _, productID := range input.ProductIDs {
		err := collectionService.RemoveProductFromCollection(c.Request.Context(), collectionId, productID)
		if err != nil {
			lastErr = err // Store last error
			fmt.Printf("Warning: Failed to remove product %s from collection %s: %v\n", productID, collectionId, err)
		} else {
			removedCount++
		}
	}

	if lastErr != nil && removedCount == 0 { // If no products were removed successfully
		if lastErr == services.ErrCollectionNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "Collection not found")
		} else {
			// Don't error if product wasn't in collection, maybe?
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to remove any products from collection: "+lastErr.Error())
		}
		return
	}

	// NOTE: Cannot efficiently get final product count without another DB call.
	// Returning a placeholder or omitting.
	finalProductCount := -1 // Indicate count might be inaccurate without fetching
	// Ideally: finalProductCount, _ := collectionService.CountCollectionProducts(...)

	// Return response as per API_DESIGN spec
	response := gin.H{
		"success":      true,
		"collectionId": collectionId,
		"productCount": finalProductCount, // Return estimated/placeholder count
	}

	utils.JSONResponse(c, http.StatusOK, response)
}
