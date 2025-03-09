package handlers

import (
	"encoding/json"
	"fmt"
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

	// Check if we have additional path parts
	if len(productIdParts) > 1 {
		// Handle variants
		if productIdParts[1] == "variants" {
			HandleProductVariants(w, r, productId, productIdParts)
			return
		}
		// Handle images
		if productIdParts[1] == "images" {
			HandleProductImages(w, r, productId, productIdParts)
			return
		}
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

// HandleProductVariants handles variant-related operations for a product
func HandleProductVariants(w http.ResponseWriter, r *http.Request, productId string, pathParts []string) {
	// List variants
	if r.Method == http.MethodGet && len(pathParts) == 2 {
		ListProductVariants(w, r, productId)
		return
	}

	// Create variant
	if r.Method == http.MethodPost && len(pathParts) == 2 {
		CreateProductVariant(w, r, productId)
		return
	}

	// Handle operations on a specific variant
	if len(pathParts) >= 3 && pathParts[2] != "" {
		variantId := pathParts[2]

		// Check if we're associating an image with a variant
		if len(pathParts) > 3 && pathParts[3] == "images" {
			HandleVariantImages(w, r, productId, variantId, pathParts)
			return
		}

		switch r.Method {
		case http.MethodPut:
			UpdateProductVariant(w, r, productId, variantId)
			return
		case http.MethodDelete:
			DeleteProductVariant(w, r, productId, variantId)
			return
		}
	}

	utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
}

// HandleProductImages handles image-related operations for a product
func HandleProductImages(w http.ResponseWriter, r *http.Request, productId string, pathParts []string) {
	// Upload image
	if r.Method == http.MethodPost && len(pathParts) == 2 {
		UploadProductImage(w, r, productId)
		return
	}

	utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
}

// HandleVariantImages handles image-related operations for a product variant
func HandleVariantImages(w http.ResponseWriter, r *http.Request, productId string, variantId string, pathParts []string) {
	// Associate image with variant
	if r.Method == http.MethodPost && len(pathParts) == 4 {
		AssociateImageWithVariant(w, r, productId, variantId)
		return
	}

	utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
}

// CreateProductVariant handles a request to create a new product variant
func CreateProductVariant(w http.ResponseWriter, r *http.Request, productId string) {
	// Decode the request body
	var input models.ProductVariantInput
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if productService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Product service not initialized")
		return
	}

	// Validate input
	if input.Title == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Variant title is required")
		return
	}

	// Create a new variant
	variant := models.ProductVariant{
		ID:              generateProductID(),
		ProductID:       productId,
		Title:           input.Title,
		SKU:             input.SKU,
		Price:           input.Price,
		Inventory:       input.Inventory,
		Options:         input.Options,
		Dimensions:      input.Dimensions,
		FulfillmentData: input.FulfillmentData,
	}

	// Get the product to set additional fields
	product, err := productService.GetProduct(r.Context(), productId)
	if err != nil {
		if err == services.ErrProductNotFound {
			utils.ErrorResponse(w, http.StatusNotFound, "Product not found")
		} else {
			utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to fetch product")
		}
		return
	}

	variant.ProductTitle = product.Title

	// Add the variant to the product
	err = productService.AddProductVariant(r.Context(), productId, &variant)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to create variant")
		return
	}

	utils.JSONResponse(w, http.StatusCreated, variant)
}

// UpdateProductVariant handles a request to update a product variant
func UpdateProductVariant(w http.ResponseWriter, r *http.Request, productId string, variantId string) {
	// Decode the request body
	var input models.ProductVariantInput
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if productService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Product service not initialized")
		return
	}

	// Validate input
	if input.Title == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Variant title is required")
		return
	}

	// Get the product to access variants
	product, err := productService.GetProduct(r.Context(), productId)
	if err != nil {
		if err == services.ErrProductNotFound {
			utils.ErrorResponse(w, http.StatusNotFound, "Product not found")
		} else {
			utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to fetch product")
		}
		return
	}

	// Find the variant
	var existingVariant *models.ProductVariant
	for i := range product.Variants {
		if product.Variants[i].ID == variantId {
			existingVariant = &product.Variants[i]
			break
		}
	}

	if existingVariant == nil {
		utils.ErrorResponse(w, http.StatusNotFound, "Variant not found")
		return
	}

	// Update the variant fields
	variant := models.ProductVariant{
		ID:              variantId,
		ProductID:       productId,
		ProductTitle:    product.Title,
		Title:           input.Title,
		SKU:             input.SKU,
		Price:           input.Price,
		Inventory:       input.Inventory,
		Options:         input.Options,
		Dimensions:      input.Dimensions,
		FulfillmentData: input.FulfillmentData,
	}

	// Update the variant
	err = productService.UpdateProductVariant(r.Context(), productId, &variant)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to update variant")
		return
	}

	utils.JSONResponse(w, http.StatusOK, variant)
}

// DeleteProductVariant handles a request to delete a product variant
func DeleteProductVariant(w http.ResponseWriter, r *http.Request, productId string, variantId string) {
	if productService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Product service not initialized")
		return
	}

	// Delete the variant
	err := productService.DeleteProductVariant(r.Context(), productId, variantId)
	if err != nil {
		if err == services.ErrProductNotFound {
			utils.ErrorResponse(w, http.StatusNotFound, "Product not found")
		} else if err == services.ErrVariantNotFound {
			utils.ErrorResponse(w, http.StatusNotFound, "Variant not found")
		} else {
			utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to delete variant")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UploadProductImage handles a request to upload an image for a product
func UploadProductImage(w http.ResponseWriter, r *http.Request, productId string) {
	// Parse the multipart form data
	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Failed to parse form")
		return
	}

	// Get the file
	file, handler, err := r.FormFile("image")
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "No image file provided")
		return
	}
	defer file.Close()

	// Get the alt text
	altText := r.FormValue("altText")

	if productService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Product service not initialized")
		return
	}

	// Get the product to verify it exists
	_, err = productService.GetProduct(r.Context(), productId)
	if err != nil {
		if err == services.ErrProductNotFound {
			utils.ErrorResponse(w, http.StatusNotFound, "Product not found")
		} else {
			utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to fetch product")
		}
		return
	}

	// In a real implementation, we would upload the image to storage
	// For now, we'll simulate it
	imageUrl := "https://cdn.lemnispace.com/images/" + handler.Filename
	image := models.Image{
		ID:        generateProductID(),
		URL:       imageUrl,
		AltText:   altText,
		Width:     1200, // Sample values
		Height:    800,
		IsDefault: true,
		Position:  1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add the image to the product
	err = productService.AddProductImage(r.Context(), productId, &image)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to add image to product")
		return
	}

	utils.JSONResponse(w, http.StatusCreated, image)
}

// AssociateImageWithVariant handles a request to associate an image with a variant
func AssociateImageWithVariant(w http.ResponseWriter, r *http.Request, productId string, variantId string) {
	// Decode the request body
	var input struct {
		ImageID string `json:"imageId"`
	}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if input.ImageID == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Image ID is required")
		return
	}

	if productService == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Product service not initialized")
		return
	}

	// Associate the image with the variant
	err = productService.AssociateImageWithVariant(r.Context(), productId, variantId, input.ImageID)
	if err != nil {
		if err == services.ErrProductNotFound {
			utils.ErrorResponse(w, http.StatusNotFound, "Product not found")
		} else if err == services.ErrVariantNotFound {
			utils.ErrorResponse(w, http.StatusNotFound, "Variant not found")
		} else if err == services.ErrImageNotFound {
			utils.ErrorResponse(w, http.StatusNotFound, "Image not found")
		} else {
			utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to associate image with variant")
		}
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"variantId": variantId,
		"imageId":   input.ImageID,
	})
}

// Update ListAllProducts to match API_DESIGN.md response format
func ListAllProducts(w http.ResponseWriter, r *http.Request) {
	utils.DebugLog("Listing all products")

	// Parse pagination parameters
	limit, cursor := getPaginationParams(r)
	utils.DebugLog("Pagination params - limit: %d, cursor: %s", limit, cursor)

	// Parse query parameters for filtering and sorting
	queryParams := r.URL.Query()
	filters := buildFilterParams(queryParams)
	sortKey, sortOrder := getSortParams(queryParams)
	utils.DebugLog("Filter and sort params - filters: %v, sortKey: %s, sortOrder: %s", filters, sortKey, sortOrder)

	if productService == nil {
		utils.ErrorLog("Product service not initialized in ListAllProducts handler")
		utils.ErrorResponse(w, http.StatusInternalServerError, "Product service not initialized")
		return
	}

	// Get products from service
	result, err := productService.ListProducts(r.Context(), limit, cursor, filters, sortKey, sortOrder)
	if err != nil {
		utils.ErrorLog("Failed to fetch products: %v", err)
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to fetch products")
		return
	}

	// Create self link
	selfLink := fmt.Sprintf("/v1/products")
	if len(queryParams) > 0 {
		selfLink += "?" + queryParams.Encode()
	}

	// Create next link if there's a next cursor
	var nextLink string
	if result.NextCursor != "" {
		// Create a new query with the next cursor
		nextQueryValues := r.URL.Query()
		nextQueryValues.Set("cursor", result.NextCursor)

		nextLink = fmt.Sprintf("/v1/products?%s", nextQueryValues.Encode())
	}

	utils.DebugLog("Found %d products", len(result.Products))

	// Format response according to API spec and test expectations
	response := struct {
		Items []models.Product       `json:"items"`
		Links models.PaginationLinks `json:"links"`
	}{
		Items: result.Products,
		Links: models.PaginationLinks{
			Self: selfLink,
			Next: nextLink,
		},
	}

	utils.DebugLog("Returning products response with links - self: %s, next: %s",
		response.Links.Self, response.Links.Next)
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
	utils.DebugLog("Deleting product with ID: %s", productId)

	if productService == nil {
		utils.ErrorLog("Product service not initialized in DeleteProduct handler")
		utils.ErrorResponse(w, http.StatusInternalServerError, "Product service not initialized")
		return
	}

	// Delete the product
	err := productService.DeleteProduct(r.Context(), productId)
	if err != nil {
		if err == services.ErrProductNotFound {
			utils.DebugLog("Product not found for deletion: %s", productId)
			utils.ErrorResponse(w, http.StatusNotFound, "Product not found")
		} else {
			utils.ErrorLog("Failed to delete product %s: %v", productId, err)
			utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to delete product")
		}
		return
	}

	// Return 204 No Content on successful deletion
	utils.DebugLog("Successfully deleted product: %s", productId)
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
