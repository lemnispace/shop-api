package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/internal/utils"
)

var (
	ErrProductNotFound = errors.New("product not found")
	ErrVariantNotFound = errors.New("variant not found")
	ErrImageNotFound   = errors.New("image not found")
)

// ProductService defines the interface for product operations
type ProductService interface {
	GetProduct(ctx context.Context, id string) (*models.Product, error)
	CreateProduct(ctx context.Context, product *models.Product) error
	UpdateProduct(ctx context.Context, product *models.Product) error
	DeleteProduct(ctx context.Context, id string) error
	ListProducts(ctx context.Context, limit int, cursor string, filters map[string]interface{}, sortKey, sortOrder string) (*ProductListResult, error)
	CountProducts(ctx context.Context, filters map[string]interface{}) (int, error)
	ListProductVariants(ctx context.Context, productID string, limit int, cursor string) ([]models.ProductVariant, string, error)
	ListAllVariants(ctx context.Context, limit int, cursor string, filters map[string]interface{}, sortKey, sortOrder string) ([]models.ProductVariant, string, error)

	// New methods for variant management
	AddProductVariant(ctx context.Context, productID string, variant *models.ProductVariant) error
	UpdateProductVariant(ctx context.Context, productID string, variant *models.ProductVariant) error
	DeleteProductVariant(ctx context.Context, productID string, variantID string) error

	// New methods for image handling
	AddProductImage(ctx context.Context, productID string, image *models.Image) error
	AssociateImageWithVariant(ctx context.Context, productID string, variantID string, imageID string) error
}

// ProductListResult represents the result of a product list operation with pagination
type ProductListResult struct {
	Products   []models.Product
	NextCursor string
}

// DynamoDBProductService is an implementation of ProductService using DynamoDB
type DynamoDBProductService struct {
	db                  *dynamodb.Client
	tableName           string
	scanLimitMultiplier int32 // Multiplier for scan limit to account for filtering
}

// NewProductService creates a new DynamoDB product service
func NewProductService(db *dynamodb.Client, tableName string) *DynamoDBProductService {
	if db == nil {
		log.Printf("WARNING: DynamoDB client is nil in NewProductService")
	}
	if tableName == "" {
		log.Printf("WARNING: Empty table name in NewProductService")
		tableName = "ShopAPI" // Default table name
	}

	// Default scan limit multiplier
	// This is used to scan more items than requested to account for filtering
	// Can be configured via environment variable PRODUCT_SCAN_MULTIPLIER
	scanLimitMultiplier := int32(100)
	if multiplierStr := os.Getenv("PRODUCT_SCAN_MULTIPLIER"); multiplierStr != "" {
		if multiplier, err := strconv.ParseInt(multiplierStr, 10, 32); err == nil && multiplier > 0 {
			scanLimitMultiplier = int32(multiplier)
			log.Printf("Using custom scan limit multiplier: %d", scanLimitMultiplier)
		} else {
			log.Printf("WARNING: Invalid PRODUCT_SCAN_MULTIPLIER value '%s', using default: %d", multiplierStr, scanLimitMultiplier)
		}
	}

	log.Printf("Initializing DynamoDB Product Service with table: %s, scan multiplier: %d", tableName, scanLimitMultiplier)

	return &DynamoDBProductService{
		db:                  db,
		tableName:           tableName,
		scanLimitMultiplier: scanLimitMultiplier,
	}
}

// GetProduct retrieves a product by ID
func (s *DynamoDBProductService) GetProduct(ctx context.Context, id string) (*models.Product, error) {
	utils.DebugLog("Getting product with ID: %s", id)

	if s.db == nil {
		utils.ErrorLog("DynamoDB client is nil in GetProduct")
		return nil, fmt.Errorf("dynamoDB client not initialized")
	}

	if id == "" {
		utils.ErrorLog("Empty product ID provided to GetProduct")
		return nil, fmt.Errorf("product ID cannot be empty")
	}

	// Get keys using the service function instead of utils
	pk, sk := ProductKey(id)
	utils.DebugLog("Using product keys - PK: %s, SK: %s", pk, sk)

	// Perform the GetItem operation
	result, err := s.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		utils.ErrorLog("Failed to get product from DynamoDB: %v", err)
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	if result.Item == nil || len(result.Item) == 0 {
		utils.ErrorLog("Product not found with ID: %s", id)
		return nil, ErrProductNotFound
	}

	// Try to unmarshal directly first
	var product models.Product
	err = attributevalue.UnmarshalMap(result.Item, &product)

	// If direct unmarshal fails, try to extract from the Data field if present
	if err != nil || product.Title == "" {
		utils.DebugLog("Direct unmarshal failed or incomplete, trying Data field: %v", err)

		// Extract the Data field which contains the serialized product
		dataAttr, ok := result.Item["Data"]
		if !ok {
			utils.ErrorLog("Product item does not contain Data field")
			return nil, fmt.Errorf("invalid product data format")
		}

		dataBytes, ok := dataAttr.(*types.AttributeValueMemberB)
		if !ok {
			utils.ErrorLog("Product Data field is not binary data")
			return nil, fmt.Errorf("invalid product data type")
		}

		if err := json.Unmarshal(dataBytes.Value, &product); err != nil {
			utils.ErrorLog("Failed to unmarshal product data: %v", err)
			return nil, fmt.Errorf("failed to unmarshal product: %w", err)
		}
	}

	// Ensure ID is set
	if product.ID == "" {
		product.ID = id
	}

	utils.DebugLog("Successfully retrieved product: %s - %s", product.ID, product.Title)
	return &product, nil
}

// CreateProduct creates a new product
func (s *DynamoDBProductService) CreateProduct(ctx context.Context, product *models.Product) error {
	utils.DebugLog("Creating product: %s", product.Title)

	if s.db == nil {
		utils.ErrorLog("DynamoDB client is nil in CreateProduct")
		return fmt.Errorf("dynamoDB client not initialized")
	}

	// Generate ID if not provided
	if product.ID == "" {
		product.ID = fmt.Sprintf("prod_%d", time.Now().UnixNano())
		utils.DebugLog("Generated product ID: %s", product.ID)
	}

	// Generate IDs for variants if not provided
	for i := range product.Variants {
		if product.Variants[i].ID == "" {
			product.Variants[i].ID = fmt.Sprintf("var_%d", time.Now().UnixNano()+int64(i))
			utils.DebugLog("Generated variant ID: %s", product.Variants[i].ID)
		}
		// Set variant product info
		product.Variants[i].ProductID = product.ID
		product.Variants[i].ProductTitle = product.Title
	}

	// Set timestamps
	now := time.Now()
	product.CreatedAt = now
	product.UpdatedAt = now

	// Marshal product data
	data, err := json.Marshal(product)
	if err != nil {
		utils.ErrorLog("Failed to marshal product: %v", err)
		return fmt.Errorf("failed to marshal product: %w", err)
	}

	// Get keys using the service function
	pk, sk := ProductKey(product.ID)
	utils.DebugLog("Using product keys - PK: %s, SK: %s", pk, sk)

	// Create item
	item := map[string]types.AttributeValue{
		"PK":         &types.AttributeValueMemberS{Value: pk},
		"SK":         &types.AttributeValueMemberS{Value: sk},
		"GSI1PK":     &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#STATUS#%s", EntityProduct, product.Status)},
		"GSI1SK":     &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#%s", EntityProduct, product.ID)},
		"EntityType": &types.AttributeValueMemberS{Value: EntityProduct},
		"Data":       &types.AttributeValueMemberB{Value: data},
		"CreatedAt":  &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		"UpdatedAt":  &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
	}

	// Add SKU as GSI2PK if available
	if product.SKU != "" {
		item["GSI2PK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("SKU#%s", product.SKU)}
		item["GSI2SK"] = &types.AttributeValueMemberS{Value: pk}
	}

	// Store product in DynamoDB
	utils.DebugLog("Storing product in DynamoDB")
	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})

	if err != nil {
		utils.ErrorLog("Failed to store product in DynamoDB: %v", err)
		return fmt.Errorf("failed to store product: %w", err)
	}

	utils.DebugLog("Successfully created product with ID: %s", product.ID)
	return nil
}

// UpdateProduct updates an existing product
func (s *DynamoDBProductService) UpdateProduct(ctx context.Context, product *models.Product) error {
	utils.DebugLog("Updating product with ID: %s", product.ID)

	if s.db == nil {
		utils.ErrorLog("DynamoDB client is nil in UpdateProduct")
		return fmt.Errorf("dynamoDB client not initialized")
	}

	if product.ID == "" {
		utils.ErrorLog("Empty product ID provided to UpdateProduct")
		return fmt.Errorf("product ID cannot be empty")
	}

	// Check if product exists
	existingProduct, err := s.GetProduct(ctx, product.ID)
	if err != nil {
		utils.ErrorLog("Failed to find product to update: %v", err)
		return fmt.Errorf("cannot update product: %w", err)
	}

	// Set timestamps
	product.CreatedAt = existingProduct.CreatedAt
	product.UpdatedAt = time.Now()

	// Marshal product data
	data, err := json.Marshal(product)
	if err != nil {
		utils.ErrorLog("Failed to marshal product: %v", err)
		return fmt.Errorf("failed to marshal product: %w", err)
	}

	// Get keys using the service function
	pk, sk := ProductKey(product.ID)
	utils.DebugLog("Using product keys - PK: %s, SK: %s", pk, sk)

	// Create item
	item := map[string]types.AttributeValue{
		"PK":         &types.AttributeValueMemberS{Value: pk},
		"SK":         &types.AttributeValueMemberS{Value: sk},
		"GSI1PK":     &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#STATUS#%s", EntityProduct, product.Status)},
		"GSI1SK":     &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#%s", EntityProduct, product.ID)},
		"EntityType": &types.AttributeValueMemberS{Value: EntityProduct},
		"Data":       &types.AttributeValueMemberB{Value: data},
		"CreatedAt":  &types.AttributeValueMemberS{Value: product.CreatedAt.Format(time.RFC3339)},
		"UpdatedAt":  &types.AttributeValueMemberS{Value: product.UpdatedAt.Format(time.RFC3339)},
	}

	// Add SKU as GSI2PK if available
	if product.SKU != "" {
		item["GSI2PK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("SKU#%s", product.SKU)}
		item["GSI2SK"] = &types.AttributeValueMemberS{Value: pk}
	}

	// Store product in DynamoDB
	utils.DebugLog("Storing updated product in DynamoDB")
	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})

	if err != nil {
		utils.ErrorLog("Failed to update product in DynamoDB: %v", err)
		return fmt.Errorf("failed to update product: %w", err)
	}

	utils.DebugLog("Successfully updated product with ID: %s", product.ID)
	return nil
}

// DeleteProduct deletes a product by ID
func (s *DynamoDBProductService) DeleteProduct(ctx context.Context, id string) error {
	utils.DebugLog("Deleting product with ID: %s", id)

	if s.db == nil {
		utils.ErrorLog("DynamoDB client is nil in DeleteProduct")
		return fmt.Errorf("dynamoDB client not initialized")
	}

	if id == "" {
		utils.ErrorLog("Empty product ID provided to DeleteProduct")
		return fmt.Errorf("product ID cannot be empty")
	}

	// Get keys using the service function
	pk, sk := ProductKey(id)
	utils.DebugLog("Using product keys - PK: %s, SK: %s", pk, sk)

	// Check if product exists first using a simplified check
	exists, err := s.productExists(ctx, id)
	if err != nil {
		utils.ErrorLog("Error checking if product exists: %v", err)
		return fmt.Errorf("error checking product existence: %w", err)
	}

	if !exists {
		utils.ErrorLog("Product not found with ID: %s", id)
		return ErrProductNotFound
	}

	// Delete product - this is a simpler operation that's less likely to hang
	_, err = s.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		utils.ErrorLog("Failed to delete product from DynamoDB: %v", err)
		return fmt.Errorf("failed to delete product: %w", err)
	}

	utils.DebugLog("Successfully deleted product with ID: %s", id)
	return nil
}

// Helper method to check if a product exists - simplified to avoid potential hanging
func (s *DynamoDBProductService) productExists(ctx context.Context, id string) (bool, error) {
	utils.DebugLog("Checking if product exists: %s", id)

	if id == "" {
		return false, fmt.Errorf("product ID cannot be empty")
	}

	// Get keys using the service function
	pk, sk := ProductKey(id)

	// Using ProjectionExpression to minimize data transfer
	result, err := s.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
		ProjectionExpression: aws.String("PK"), // Only retrieve the PK attribute
	})

	if err != nil {
		utils.ErrorLog("Error checking product existence: %v", err)
		return false, err
	}

	return result.Item != nil && len(result.Item) > 0, nil
}

// ListProducts lists products from DynamoDB with pagination, filtering, and sorting
func (s *DynamoDBProductService) ListProducts(ctx context.Context, limit int, cursor string, filters map[string]interface{}, sortKey, sortOrder string) (*ProductListResult, error) {
	utils.DebugLog("Listing products with limit: %d, cursor: %s, filters: %v, sort: %s %s",
		limit, cursor, filters, sortKey, sortOrder)

	// TODO(perf): Replace the table-wide Scan + in-memory filter/sort with GSIs that support the
	// common filters (status, collection, createdAt) so pagination happens server-side and large
	// catalogs do not hit the 1 MB scan cap every request.
	if s.db == nil {
		utils.ErrorLog("DynamoDB client is nil in ListProducts")
		return nil, fmt.Errorf("dynamoDB client not initialized")
	}

	if limit <= 0 {
		limit = 20 // Default limit
	}

	// If a collection filter is present, pre-fetch the allowed product IDs
	var allowedProductIDs map[string]bool
	if collectionID, ok := filters["collection"].(string); ok && collectionID != "" {
		utils.DebugLog("Collection filter detected: %s - fetching allowed product IDs", collectionID)
		var err error
		allowedProductIDs, err = s.getCollectionProductIDs(ctx, collectionID)
		if err != nil {
			utils.ErrorLog("Failed to get collection product IDs: %v", err)
			// Return error - we can't properly filter without collection data
			return nil, fmt.Errorf("failed to get products for collection %s: %w", collectionID, err)
		}
		utils.DebugLog("Collection %s has %d products", collectionID, len(allowedProductIDs))
	}

	// Use a scan with filter expressions to find products
	// Note: Since FilterExpression is applied AFTER Limit, we need to scan more items
	// to ensure we get enough products after filtering. This is a known limitation.
	// TODO: Use GSI for better performance (see comment on line 378)
	scanLimit := int32(limit) * s.scanLimitMultiplier
	if scanLimit > 1000 {
		scanLimit = 1000 // DynamoDB max
	}

	scanInput := &dynamodb.ScanInput{
		TableName:        aws.String(s.tableName),
		Limit:            aws.Int32(scanLimit),
		FilterExpression: aws.String("begins_with(PK, :pk) AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#", EntityProduct)},
			":sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#", EntityProduct)},
		},
	}

	// Add pagination if cursor is provided
	if cursor != "" {
		utils.DebugLog("Decoding cursor: %s", cursor)
		exclusiveStartKey, err := utils.DecodeCursor(cursor)
		if err != nil {
			utils.ErrorLog("Failed to decode cursor: %v", err)
			return nil, fmt.Errorf("invalid pagination cursor: %w", err)
		}
		scanInput.ExclusiveStartKey = exclusiveStartKey
	}

	utils.DebugLog("Executing DynamoDB scan: %+v", scanInput)

	// Execute scan
	result, err := s.db.Scan(ctx, scanInput)
	if err != nil {
		utils.ErrorLog("DynamoDB scan failed: %v", err)
		return nil, fmt.Errorf("failed to query products: %w", err)
	}

	utils.DebugLog("Scan returned %d items", len(result.Items))

	// Unmarshal results
	var products []models.Product
	for _, item := range result.Items {
		var product models.Product

		// First try to get the product from the Data field if it exists
		dataAttr, ok := item["Data"]
		if ok {
			dataBytes, ok := dataAttr.(*types.AttributeValueMemberB)
			if ok {
				if err := json.Unmarshal(dataBytes.Value, &product); err != nil {
					utils.ErrorLog("Failed to unmarshal product data: %v", err)
					continue
				}
			}
		} else {
			// If no Data field, try direct unmarshal
			if err := attributevalue.UnmarshalMap(item, &product); err != nil {
				utils.ErrorLog("Failed to unmarshal product: %v", err)
				continue
			}
		}

		// Extract ID from PK if not set directly
		if product.ID == "" && item["PK"] != nil {
			if pk, ok := item["PK"].(*types.AttributeValueMemberS); ok {
				parts := strings.Split(pk.Value, "#")
				if len(parts) > 1 {
					product.ID = parts[1]
				}
			}
		}

		utils.DebugLog("Found product: %s - %s", product.ID, product.Title)
		products = append(products, product)
	}

	// Apply in-memory filtering
	filteredProducts := []models.Product{}
	for _, product := range products {
		if matchesFilters(product, filters, allowedProductIDs) {
			filteredProducts = append(filteredProducts, product)
		}
	}

	// Sort products
	sortProducts(filteredProducts, sortKey, sortOrder)

	// Get next page cursor
	var nextCursor string
	if result.LastEvaluatedKey != nil && len(result.LastEvaluatedKey) > 0 {
		utils.DebugLog("Generating next cursor from LastEvaluatedKey")
		nextCursor, err = utils.EncodeCursor(result.LastEvaluatedKey)
		if err != nil {
			utils.ErrorLog("Failed to encode cursor: %v", err)
			// Continue without cursor
		}
	}

	utils.DebugLog("Returning %d products with nextCursor: %s", len(filteredProducts), nextCursor)

	return &ProductListResult{
		Products:   filteredProducts,
		NextCursor: nextCursor,
	}, nil
}

// CountProducts returns the count of products based on filters
// Note: This implementation fetches and filters products in-memory because DynamoDB
// doesn't support filtered counts efficiently without GSIs for each filter combination
func (s *DynamoDBProductService) CountProducts(ctx context.Context, filters map[string]interface{}) (int, error) {
	utils.DebugLog("Counting products with filters: %v", filters)

	if s.db == nil {
		utils.ErrorLog("DynamoDB client is nil in CountProducts")
		return 0, fmt.Errorf("dynamoDB client not initialized")
	}

	// If a collection filter is present, pre-fetch the allowed product IDs
	var allowedProductIDs map[string]bool
	if collectionID, ok := filters["collection"].(string); ok && collectionID != "" {
		utils.DebugLog("Collection filter detected for count: %s - fetching allowed product IDs", collectionID)
		var err error
		allowedProductIDs, err = s.getCollectionProductIDs(ctx, collectionID)
		if err != nil {
			utils.ErrorLog("Failed to get collection product IDs for count: %v", err)
			// Return error - we can't properly count without collection data
			return 0, fmt.Errorf("failed to get products for collection %s: %w", collectionID, err)
		}
		utils.DebugLog("Collection %s has %d products for count", collectionID, len(allowedProductIDs))
	}

	// If no filters, use simple count
	if len(filters) == 0 {
		scanInput := &dynamodb.ScanInput{
			TableName:        aws.String(s.tableName),
			FilterExpression: aws.String("begins_with(PK, :pk) AND begins_with(SK, :sk)"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#", EntityProduct)},
				":sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#", EntityProduct)},
			},
			Select: types.SelectCount,
		}

		result, err := s.db.Scan(ctx, scanInput)
		if err != nil {
			utils.ErrorLog("DynamoDB count scan failed: %v", err)
			return 0, fmt.Errorf("failed to count products: %w", err)
		}

		return int(result.Count), nil
	}

	// For filtered counts, we need to fetch all products and apply filters
	// This is inefficient but necessary without proper GSIs
	scanInput := &dynamodb.ScanInput{
		TableName:        aws.String(s.tableName),
		FilterExpression: aws.String("begins_with(PK, :pk) AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#", EntityProduct)},
			":sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#", EntityProduct)},
		},
	}

	result, err := s.db.Scan(ctx, scanInput)
	if err != nil {
		utils.ErrorLog("DynamoDB count scan failed: %v", err)
		return 0, fmt.Errorf("failed to count products: %w", err)
	}

	// Unmarshal and filter products
	count := 0
	for _, item := range result.Items {
		var product models.Product

		// Try to unmarshal from Data field
		dataAttr, ok := item["Data"]
		if ok {
			dataBytes, ok := dataAttr.(*types.AttributeValueMemberB)
			if ok {
				if err := json.Unmarshal(dataBytes.Value, &product); err != nil {
					continue
				}
			}
		} else {
			// Try direct unmarshal
			if err := attributevalue.UnmarshalMap(item, &product); err != nil {
				continue
			}
		}

		// Extract ID from PK if not set directly
		if product.ID == "" && item["PK"] != nil {
			if pk, ok := item["PK"].(*types.AttributeValueMemberS); ok {
				parts := strings.Split(pk.Value, "#")
				if len(parts) > 1 {
					product.ID = parts[1]
				}
			}
		}

		// Apply filters
		if matchesFilters(product, filters, allowedProductIDs) {
			count++
		}
	}

	utils.DebugLog("Filtered count: %d products match filters", count)
	return count, nil
}

// ListProductVariants lists variants for a specific product with pagination
func (s *DynamoDBProductService) ListProductVariants(ctx context.Context, productID string, limit int, cursor string) ([]models.ProductVariant, string, error) {
	// Check if product exists
	product, err := s.GetProduct(ctx, productID)
	if err != nil {
		return nil, "", err
	}

	// Extract variants with pagination
	totalVariants := len(product.Variants)
	if totalVariants == 0 {
		return []models.ProductVariant{}, "", nil
	}

	// Parse cursor
	startIndex := 0
	if cursor != "" {
		startIndex, err = strconv.Atoi(cursor)
		if err != nil {
			return nil, "", err
		}
	}

	// Calculate end index
	endIndex := startIndex + limit
	if endIndex > totalVariants {
		endIndex = totalVariants
	}

	// Extract variants
	variants := product.Variants[startIndex:endIndex]

	// Calculate next cursor
	var nextCursor string
	if endIndex < totalVariants {
		nextCursor = strconv.Itoa(endIndex)
	}

	return variants, nextCursor, nil
}

// ListAllVariants lists all variants across products with pagination and filtering
func (s *DynamoDBProductService) ListAllVariants(ctx context.Context, limit int, cursor string, filters map[string]interface{}, sortKey, sortOrder string) ([]models.ProductVariant, string, error) {
	// This would typically use a GSI to efficiently query variants
	// TODO(perf): Page variants directly from DynamoDB (e.g., dedicated GSI) and honor the incoming
	// cursor/limit instead of always loading the first 100 products into memory.
	// For now, we'll use a simplified approach of getting all products and filtering
	result, err := s.ListProducts(ctx, 100, "", filters, sortKey, sortOrder)
	if err != nil {
		return nil, "", err
	}

	allVariants := []models.ProductVariant{}
	for _, product := range result.Products {
		for _, variant := range product.Variants {
			allVariants = append(allVariants, variant)
		}
	}

	// Apply pagination
	startIndex := 0
	if cursor != "" {
		var err error
		startIndex, err = strconv.Atoi(cursor)
		if err != nil {
			startIndex = 0
		}
	}

	endIndex := startIndex + limit
	if endIndex > len(allVariants) {
		endIndex = len(allVariants)
	}

	var resultVariants []models.ProductVariant
	if startIndex < len(allVariants) {
		resultVariants = allVariants[startIndex:endIndex]
	} else {
		resultVariants = []models.ProductVariant{}
	}

	// Create cursor for next page
	var nextCursor string
	if endIndex < len(allVariants) {
		nextCursor = strconv.Itoa(endIndex)
	}

	return resultVariants, nextCursor, nil
}

// Helper methods

// getCollectionProductIDs retrieves all product IDs that belong to a specific collection
// This method handles DynamoDB pagination to ensure all products are retrieved, even for
// collections with more than 1 MB of relationship data
func (s *DynamoDBProductService) getCollectionProductIDs(ctx context.Context, collectionID string) (map[string]bool, error) {
	utils.DebugLog("Getting product IDs for collection: %s", collectionID)

	// Query for all product relationships in the collection
	pk := fmt.Sprintf("%s#%s", EntityCollection, collectionID)
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk_prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":        &types.AttributeValueMemberS{Value: pk},
			":sk_prefix": &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#", EntityProduct)},
		},
	}

	productIDs := make(map[string]bool)
	var lastEvaluatedKey map[string]types.AttributeValue
	pageCount := 0

	// Loop through all pages until we've fetched all collection-product relationships
	for {
		pageCount++
		utils.DebugLog("Fetching page %d for collection %s", pageCount, collectionID)

		// Set the starting point for this page
		if lastEvaluatedKey != nil {
			queryInput.ExclusiveStartKey = lastEvaluatedKey
		}

		result, err := s.db.Query(ctx, queryInput)
		if err != nil {
			utils.ErrorLog("Failed to query collection products (page %d): %v", pageCount, err)
			return nil, fmt.Errorf("failed to query collection products: %w", err)
		}

		// Extract product IDs from SK values for this page
		for _, item := range result.Items {
			skAttr, ok := item["SK"]
			if !ok {
				continue
			}
			sk, ok := skAttr.(*types.AttributeValueMemberS)
			if !ok {
				continue
			}

			// Extract product ID from SK (format: PRODUCT#<productID>)
			parts := strings.Split(sk.Value, "#")
			if len(parts) >= 2 {
				productID := parts[1]
				productIDs[productID] = true
				utils.DebugLog("Found product %s in collection %s (page %d)", productID, collectionID, pageCount)
			}
		}

		// Check if there are more pages to fetch
		lastEvaluatedKey = result.LastEvaluatedKey
		if lastEvaluatedKey == nil {
			// No more pages - we've fetched all products
			break
		}

		utils.DebugLog("Collection %s has more pages - continuing (current count: %d)", collectionID, len(productIDs))
	}

	utils.DebugLog("Collection %s contains %d products (fetched %d pages)", collectionID, len(productIDs), pageCount)
	return productIDs, nil
}

// matchesFilters checks if a product matches the filter criteria
// allowedProductIDs is an optional map used for collection filtering (can be nil if no collection filter)
func matchesFilters(product models.Product, filters map[string]interface{}, allowedProductIDs map[string]bool) bool {
	if len(filters) == 0 {
		return true
	}

	for key, value := range filters {
		switch key {
		case "status":
			if status, ok := value.(string); ok && product.Status != status {
				return false
			}
		case "collection":
			// Filter by collection - check if product belongs to the collection
			// Collection membership is determined via the allowedProductIDs map
			// which is pre-fetched by querying DynamoDB for collection-product relationships
			if collectionID, ok := value.(string); ok {
				// If allowedProductIDs is nil, it means we couldn't fetch collection products
				// In this case, exclude all products (fail-closed approach)
				if allowedProductIDs == nil {
					utils.DebugLog("Collection filter %s active but no allowed products - excluding product %s", collectionID, product.ID)
					return false
				}
				// Check if this product is in the allowed set
				if !allowedProductIDs[product.ID] {
					utils.DebugLog("Product %s not in collection %s - excluding", product.ID, collectionID)
					return false
				}
				utils.DebugLog("Product %s is in collection %s - including", product.ID, collectionID)
			}
		case "sku":
			if sku, ok := value.(string); ok && product.SKU != sku {
				return false
			}
		case "title":
			if title, ok := value.(string); ok && !contains(product.Title, title) {
				return false
			}
		case "price_min":
			if priceMinStr, ok := value.(string); ok {
				if priceMin, err := strconv.ParseFloat(priceMinStr, 64); err == nil && product.Price < priceMin {
					return false
				}
			}
		case "price_max":
			if priceMaxStr, ok := value.(string); ok {
				if priceMax, err := strconv.ParseFloat(priceMaxStr, 64); err == nil && product.Price > priceMax {
					return false
				}
			}
		case "tag":
			if tag, ok := value.(string); ok && !containsTag(product.Tags, tag) {
				return false
			} else if tags, ok := value.([]string); ok && !containsAnyTag(product.Tags, tags) {
				return false
			}
		}
	}

	return true
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// containsTag checks if a tag is in the list of tags
func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

// containsAnyTag checks if any tag in the list is in the product tags
func containsAnyTag(productTags, tags []string) bool {
	for _, tag := range tags {
		if containsTag(productTags, tag) {
			return true
		}
	}
	return false
}

// sortProducts sorts products by the given key and order
func sortProducts(products []models.Product, sortKey, sortOrder string) {
	// Simplified sort implementation
	// In a real implementation, this would be more sophisticated
	sort.Slice(products, func(i, j int) bool {
		less := false

		switch sortKey {
		case "created_at":
			less = products[i].CreatedAt.Before(products[j].CreatedAt)
		case "updated_at":
			less = products[i].UpdatedAt.Before(products[j].UpdatedAt)
		case "title":
			less = products[i].Title < products[j].Title
		case "price":
			less = products[i].Price < products[j].Price
		default:
			less = products[i].CreatedAt.Before(products[j].CreatedAt)
		}

		// Reverse for descending order
		if sortOrder == "desc" {
			return !less
		}
		return less
	})
}

// AddProductVariant adds a new variant to a product
func (s *DynamoDBProductService) AddProductVariant(ctx context.Context, productID string, variant *models.ProductVariant) error {
	// First, get the product
	product, err := s.GetProduct(ctx, productID)
	if err != nil {
		return err
	}

	// Set variant properties
	variant.ProductID = productID
	variant.ProductTitle = product.Title

	// Generate ID if not provided
	if variant.ID == "" {
		variant.ID = fmt.Sprintf("var_%d", time.Now().UnixNano())
	}

	// Add the variant to the product
	product.Variants = append(product.Variants, *variant)

	// Update the product's update timestamp
	product.UpdatedAt = time.Now()

	// Save the updated product
	err = s.UpdateProduct(ctx, product)
	if err != nil {
		return err
	}

	// Update the original variant with the ID
	*variant = product.Variants[len(product.Variants)-1]

	return nil
}

// UpdateProductVariant updates an existing variant of a product
func (s *DynamoDBProductService) UpdateProductVariant(ctx context.Context, productID string, variant *models.ProductVariant) error {
	// First, get the product
	product, err := s.GetProduct(ctx, productID)
	if err != nil {
		return err
	}

	// Find the variant
	variantIndex := -1
	for i, v := range product.Variants {
		if v.ID == variant.ID {
			variantIndex = i
			break
		}
	}

	if variantIndex == -1 {
		return ErrVariantNotFound
	}

	// Update the variant
	variant.ProductID = productID
	variant.ProductTitle = product.Title
	product.Variants[variantIndex] = *variant

	// Update the product's update timestamp
	product.UpdatedAt = time.Now()

	// Save the updated product
	err = s.UpdateProduct(ctx, product)
	if err != nil {
		return err
	}

	// Update the original variant
	*variant = product.Variants[variantIndex]

	return nil
}

// DeleteProductVariant deletes a variant from a product
func (s *DynamoDBProductService) DeleteProductVariant(ctx context.Context, productID string, variantID string) error {
	// First, get the product
	product, err := s.GetProduct(ctx, productID)
	if err != nil {
		return err
	}

	// Find the variant
	variantIndex := -1
	for i, v := range product.Variants {
		if v.ID == variantID {
			variantIndex = i
			break
		}
	}

	if variantIndex == -1 {
		return ErrVariantNotFound
	}

	// Remove the variant
	product.Variants = append(product.Variants[:variantIndex], product.Variants[variantIndex+1:]...)

	// Update the product's update timestamp
	product.UpdatedAt = time.Now()

	// Save the updated product
	return s.UpdateProduct(ctx, product)
}

// AddProductImage adds a new image to a product
func (s *DynamoDBProductService) AddProductImage(ctx context.Context, productID string, image *models.Image) error {
	// First, get the product
	product, err := s.GetProduct(ctx, productID)
	if err != nil {
		return err
	}

	// Generate ID if not provided
	if image.ID == "" {
		image.ID = fmt.Sprintf("img_%d", time.Now().UnixNano())
	}

	// Set timestamps
	now := time.Now()
	image.CreatedAt = now
	image.UpdatedAt = now

	// Add the image to the product
	product.Images = append(product.Images, *image)

	// Update the product's update timestamp
	product.UpdatedAt = now

	// Save the updated product
	err = s.UpdateProduct(ctx, product)
	if err != nil {
		return err
	}

	// Update the original image with the ID
	*image = product.Images[len(product.Images)-1]

	return nil
}

// AssociateImageWithVariant associates an image with a variant
func (s *DynamoDBProductService) AssociateImageWithVariant(ctx context.Context, productID string, variantID string, imageID string) error {
	// First, get the product
	product, err := s.GetProduct(ctx, productID)
	if err != nil {
		return err
	}

	// Find the variant
	variantFound := false
	for _, v := range product.Variants {
		if v.ID == variantID {
			variantFound = true
			break
		}
	}

	if !variantFound {
		return ErrVariantNotFound
	}

	// Find the image
	imageIndex := -1
	for i, img := range product.Images {
		if img.ID == imageID {
			imageIndex = i
			break
		}
	}

	if imageIndex == -1 {
		return ErrImageNotFound
	}

	// Associate the image with the variant
	if product.Images[imageIndex].Variants == nil {
		product.Images[imageIndex].Variants = []string{variantID}
	} else {
		// Check if already associated
		for _, vid := range product.Images[imageIndex].Variants {
			if vid == variantID {
				return nil // Already associated
			}
		}
		product.Images[imageIndex].Variants = append(product.Images[imageIndex].Variants, variantID)
	}

	// Update the product's update timestamp
	product.UpdatedAt = time.Now()

	// Save the updated product
	return s.UpdateProduct(ctx, product)
}
