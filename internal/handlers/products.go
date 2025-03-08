package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/internal/services"
	"github.com/lemnispace/shop-api/internal/utils"
)

// ProductService is a reference to the product service
var productService services.ProductService

// SetProductService sets the product service for the handlers
func SetProductService(service services.ProductService) {
	productService = service
}

// ProductsHandler handles requests to /v1/products
func ProductsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ListAllProducts(w, r)
	case http.MethodPost:
		CreateProduct(w, r)
	default:
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// ProductDetailHandler handles requests to /v1/products/{productId}
func ProductDetailHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	apiPrefix := "/v1/products/"
	if !strings.HasPrefix(path, apiPrefix) {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid path")
		return
	}

	// Extract productId from URL
	productId := strings.TrimPrefix(path, apiPrefix)
	productIdParts := strings.Split(productId, "/")
	if len(productIdParts) == 0 || productIdParts[0] == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Product ID is required")
		return
	}
	productId = productIdParts[0]

	// Check if we have a variants subpath
	if len(productIdParts) > 1 && productIdParts[1] == "variants" {
		ListProductVariants(w, r, productId)
		return
	}

	switch r.Method {
	case http.MethodGet:
		GetProduct(w, r, productId)
	case http.MethodPut:
		UpdateProduct(w, r, productId)
	case http.MethodDelete:
		DeleteProduct(w, r, productId)
	default:
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// ProductCountHandler handles requests to /v1/products/count
func ProductCountHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse query parameters for filtering
	queryParams := r.URL.Query()
	filters := buildFilterParams(queryParams)

	if productService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Product service not initialized")
		return
	}

	// Get count from service
	count, err := productService.CountProducts(r.Context(), filters)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to count products")
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"count": count,
	})
}

// ProductVariantsHandler handles requests to /v1/products/variants
func ProductVariantsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ListAllVariants(w, r)
	default:
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func ListAllProducts(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	limit, cursor := getPaginationParams(r)

	// Parse query parameters for filtering and sorting
	queryParams := r.URL.Query()
	filters := buildFilterParams(queryParams)
	sortKey, sortOrder := getSortParams(queryParams)

	// Initialize response structure
	response := models.PaginatedResponse{
		Items: []interface{}{},
		Links: models.PaginationLinks{
			Self: r.URL.String(),
		},
	}

	if productService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Product service not initialized")
		return
	}

	// Get products from service
	result, err := productService.ListProducts(r.Context(), limit, cursor, filters, sortKey, sortOrder)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to fetch products")
		return
	}

	// Populate response with products
	for _, product := range result.Products {
		response.Items = append(response.Items, product)
	}

	// Add pagination links
	if result.NextCursor != "" {
		nextURL := buildNextPageURL(r.URL, result.NextCursor)
		response.Links.Next = nextURL
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

func CreateProduct(w http.ResponseWriter, r *http.Request) {
	// Decode the request body
	var input models.ProductInput
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate the product input
	if err := validateProductInput(input); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if productService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Product service not initialized")
		return
	}

	// Create a new product instance
	product := models.Product{
		ID:              generateProductID(),
		Title:           input.Title,
		Description:     input.Description,
		Price:           input.Price,
		SKU:             input.SKU,
		Status:          input.Status,
		Inventory:       input.Inventory,
		Tags:            input.Tags,
		CustomFields:    input.CustomFields,
		Dimensions:      input.Dimensions,
		FulfillmentData: input.FulfillmentData,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Process variants
	if len(input.Variants) > 0 {
		product.Variants = make([]models.ProductVariant, 0, len(input.Variants))
		for _, variantInput := range input.Variants {
			variant := models.ProductVariant{
				ID:              generateProductID(),
				ProductID:       product.ID,
				ProductTitle:    product.Title,
				SKU:             variantInput.SKU,
				Title:           variantInput.Title,
				Price:           variantInput.Price,
				Inventory:       variantInput.Inventory,
				Options:         variantInput.Options,
				Dimensions:      variantInput.Dimensions,
				FulfillmentData: variantInput.FulfillmentData,
			}
			product.Variants = append(product.Variants, variant)
		}
	}

	// Save the product
	err = productService.CreateProduct(r.Context(), &product)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to create product")
		return
	}

	utils.JSONResponse(w, http.StatusCreated, product)
}

func GetProduct(w http.ResponseWriter, r *http.Request, productId string) {
	if productService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Product service not initialized")
		return
	}

	// Get product from service
	product, err := productService.GetProduct(r.Context(), productId)
	if err != nil {
		if err == services.ErrProductNotFound {
			utils.ErrorResponse(w, http.StatusNotFound, "Product not found")
		} else {
			utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to fetch product")
		}
		return
	}

	utils.JSONResponse(w, http.StatusOK, product)
}

func UpdateProduct(w http.ResponseWriter, r *http.Request, productId string) {
	// Decode the request body
	var input models.ProductInput
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate the product input
	if err := validateProductInput(input); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if productService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Product service not initialized")
		return
	}

	// First fetch the existing product to preserve fields not in the input
	existingProduct, err := productService.GetProduct(r.Context(), productId)
	if err != nil {
		if err == services.ErrProductNotFound {
			utils.ErrorResponse(w, http.StatusNotFound, "Product not found")
		} else {
			utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to fetch product")
		}
		return
	}

	// Update the product fields
	product := models.Product{
		ID:              productId,
		Title:           input.Title,
		Description:     input.Description,
		Price:           input.Price,
		SKU:             input.SKU,
		Status:          input.Status,
		Inventory:       input.Inventory,
		Tags:            input.Tags,
		CustomFields:    input.CustomFields,
		Dimensions:      input.Dimensions,
		FulfillmentData: input.FulfillmentData,
		CreatedAt:       existingProduct.CreatedAt,
		UpdatedAt:       time.Now(),
	}

	// Process variants
	if len(input.Variants) > 0 {
		product.Variants = make([]models.ProductVariant, 0, len(input.Variants))
		for _, variantInput := range input.Variants {
			variant := models.ProductVariant{
				ID:              generateProductID(),
				ProductID:       product.ID,
				ProductTitle:    product.Title,
				SKU:             variantInput.SKU,
				Title:           variantInput.Title,
				Price:           variantInput.Price,
				Inventory:       variantInput.Inventory,
				Options:         variantInput.Options,
				Dimensions:      variantInput.Dimensions,
				FulfillmentData: variantInput.FulfillmentData,
			}
			product.Variants = append(product.Variants, variant)
		}
	} else {
		// Preserve existing variants if none provided
		product.Variants = existingProduct.Variants
	}

	// Save the updated product
	err = productService.UpdateProduct(r.Context(), &product)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to update product")
		return
	}

	utils.JSONResponse(w, http.StatusOK, product)
}

func DeleteProduct(w http.ResponseWriter, r *http.Request, productId string) {
	if productService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Product service not initialized")
		return
	}

	// Delete the product
	err := productService.DeleteProduct(r.Context(), productId)
	if err != nil {
		if err == services.ErrProductNotFound {
			utils.ErrorResponse(w, http.StatusNotFound, "Product not found")
		} else {
			utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to delete product")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func ListProductVariants(w http.ResponseWriter, r *http.Request, productId string) {
	// Parse pagination parameters
	limit, cursor := getPaginationParams(r)

	if productService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Product service not initialized")
		return
	}

	// Get variants from service
	variants, nextCursor, err := productService.ListProductVariants(r.Context(), productId, limit, cursor)
	if err != nil {
		if err == services.ErrProductNotFound {
			utils.ErrorResponse(w, http.StatusNotFound, "Product not found")
		} else {
			utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to fetch product variants")
		}
		return
	}

	// Initialize response with pagination
	response := models.PaginatedResponse{
		Items: make([]interface{}, len(variants)),
		Links: models.PaginationLinks{
			Self: r.URL.String(),
		},
	}

	// Populate items
	for i, variant := range variants {
		response.Items[i] = variant
	}

	// Add pagination links
	if nextCursor != "" {
		nextURL := buildNextPageURL(r.URL, nextCursor)
		response.Links.Next = nextURL
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

func ListAllVariants(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	limit, cursor := getPaginationParams(r)

	// Parse query parameters for filtering and sorting
	queryParams := r.URL.Query()
	filters := buildFilterParams(queryParams)
	sortKey, sortOrder := getSortParams(queryParams)

	if productService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Product service not initialized")
		return
	}

	// Get variants from service
	variants, nextCursor, err := productService.ListAllVariants(r.Context(), limit, cursor, filters, sortKey, sortOrder)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to fetch variants")
		return
	}

	// Initialize response with pagination
	response := models.PaginatedResponse{
		Items: make([]interface{}, len(variants)),
		Links: models.PaginationLinks{
			Self: r.URL.String(),
		},
	}

	// Populate items
	for i, variant := range variants {
		response.Items[i] = variant
	}

	// Add pagination links
	if nextCursor != "" {
		nextURL := buildNextPageURL(r.URL, nextCursor)
		response.Links.Next = nextURL
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// Helper functions

func getPaginationParams(r *http.Request) (int, string) {
	// Default limit and cursor
	defaultLimit := 25
	query := r.URL.Query()

	// Parse limit
	limitStr := query.Get("limit")
	limit := defaultLimit
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			// Cap limit at 100
			if parsedLimit > 100 {
				parsedLimit = 100
			}
			limit = parsedLimit
		}
	}

	// Get cursor
	cursor := query.Get("cursor")

	return limit, cursor
}

func buildFilterParams(query map[string][]string) map[string]interface{} {
	// Initialize filters map
	filters := make(map[string]interface{})

	// Common filter parameters
	filterParams := []string{"status", "title", "price_min", "price_max", "created_at_min", "created_at_max", "updated_at_min", "updated_at_max", "tag"}

	// Process each filter parameter
	for _, param := range filterParams {
		if values, exists := query[param]; exists && len(values) > 0 {
			// Handle multi-value parameters
			if param == "tag" && len(values) > 1 {
				filters[param] = values
			} else {
				filters[param] = values[0]
			}
		}
	}

	return filters
}

func getSortParams(query map[string][]string) (string, string) {
	// Default sort key and order
	defaultSortKey := "created_at"
	defaultSortOrder := "desc"

	// Get sort key
	sortKey := defaultSortKey
	if sortKeys, exists := query["sort_key"]; exists && len(sortKeys) > 0 {
		// Validate sort key
		validSortKeys := map[string]bool{
			"created_at": true,
			"updated_at": true,
			"title":      true,
			"price":      true,
		}

		if validSortKeys[sortKeys[0]] {
			sortKey = sortKeys[0]
		}
	}

	// Get sort order
	sortOrder := defaultSortOrder
	if sortOrders, exists := query["sort_order"]; exists && len(sortOrders) > 0 {
		// Validate sort order
		if sortOrders[0] == "asc" || sortOrders[0] == "desc" {
			sortOrder = sortOrders[0]
		}
	}

	return sortKey, sortOrder
}

func buildNextPageURL(urlObj *url.URL, cursor string) string {
	query := urlObj.Query()
	query.Set("cursor", cursor)
	urlObj.RawQuery = query.Encode()
	return urlObj.String()
}

type validationError struct {
	Field   string
	Message string
}

func (e *validationError) Error() string {
	return e.Message
}

func validateProductInput(input models.ProductInput) error {
	if input.Title == "" {
		return &validationError{Field: "title", Message: "Title is required"}
	}

	if input.Price < 0 {
		return &validationError{Field: "price", Message: "Price cannot be negative"}
	}

	validStatuses := map[string]bool{
		"draft":    true,
		"active":   true,
		"archived": true,
	}

	if input.Status != "" && !validStatuses[input.Status] {
		return &validationError{Field: "status", Message: "Status must be one of: draft, active, archived"}
	}

	return nil
}

func generateProductID() string {
	// This is a simple placeholder for generating a unique ID
	// In a real implementation, this would use a more robust method
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}
