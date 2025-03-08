package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/lemnispace/shop-api/internal/models"
)

var (
	ErrProductNotFound = errors.New("product not found")
	ErrVariantNotFound = errors.New("variant not found")
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
}

// ProductListResult represents the result of a product list operation with pagination
type ProductListResult struct {
	Products   []models.Product
	NextCursor string
}

// DynamoDBProductService is an implementation of ProductService using DynamoDB
type DynamoDBProductService struct {
	db        *dynamodb.Client
	tableName string
}

// NewProductService creates a new DynamoDB product service
func NewProductService(db *dynamodb.Client, tableName string) *DynamoDBProductService {
	return &DynamoDBProductService{
		db:        db,
		tableName: tableName,
	}
}

// GetProduct retrieves a product by ID
func (s *DynamoDBProductService) GetProduct(ctx context.Context, id string) (*models.Product, error) {
	pk, sk := productKey(id)

	result, err := s.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, ErrProductNotFound
	}

	var item struct {
		Data []byte `dynamodbav:"Data"`
	}
	if err := attributevalue.UnmarshalMap(result.Item, &item); err != nil {
		return nil, err
	}

	var product models.Product
	if err := json.Unmarshal(item.Data, &product); err != nil {
		return nil, err
	}

	return &product, nil
}

// CreateProduct creates a new product
func (s *DynamoDBProductService) CreateProduct(ctx context.Context, product *models.Product) error {
	// Set timestamps
	now := time.Now()
	product.CreatedAt = now
	product.UpdatedAt = now

	// Marshal product data
	data, err := json.Marshal(product)
	if err != nil {
		return err
	}

	pk, sk := productKey(product.ID)

	// Create item
	item := map[string]types.AttributeValue{
		"PK":         &types.AttributeValueMemberS{Value: pk},
		"SK":         &types.AttributeValueMemberS{Value: sk},
		"GSI1PK":     &types.AttributeValueMemberS{Value: fmt.Sprintf("PRODUCT#STATUS#%s", product.Status)},
		"GSI1SK":     &types.AttributeValueMemberS{Value: fmt.Sprintf("PRODUCT#%s", product.ID)},
		"EntityType": &types.AttributeValueMemberS{Value: "PRODUCT"},
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
	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	return err
}

// UpdateProduct updates an existing product
func (s *DynamoDBProductService) UpdateProduct(ctx context.Context, product *models.Product) error {
	// Check if product exists
	existingProduct, err := s.GetProduct(ctx, product.ID)
	if err != nil {
		return err
	}

	// Set timestamps
	product.CreatedAt = existingProduct.CreatedAt
	product.UpdatedAt = time.Now()

	// Marshal product data
	data, err := json.Marshal(product)
	if err != nil {
		return err
	}

	pk, sk := productKey(product.ID)

	// Create item
	item := map[string]types.AttributeValue{
		"PK":         &types.AttributeValueMemberS{Value: pk},
		"SK":         &types.AttributeValueMemberS{Value: sk},
		"GSI1PK":     &types.AttributeValueMemberS{Value: fmt.Sprintf("PRODUCT#STATUS#%s", product.Status)},
		"GSI1SK":     &types.AttributeValueMemberS{Value: fmt.Sprintf("PRODUCT#%s", product.ID)},
		"EntityType": &types.AttributeValueMemberS{Value: "PRODUCT"},
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
	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	return err
}

// DeleteProduct deletes a product by ID
func (s *DynamoDBProductService) DeleteProduct(ctx context.Context, id string) error {
	pk, sk := productKey(id)

	// Check if product exists
	_, err := s.GetProduct(ctx, id)
	if err != nil {
		return err
	}

	// Delete product
	_, err = s.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	return err
}

// ListProducts lists products with pagination, filtering, and sorting
func (s *DynamoDBProductService) ListProducts(ctx context.Context, limit int, cursor string, filters map[string]interface{}, sortKey, sortOrder string) (*ProductListResult, error) {
	// For simplicity, we'll use a scan operation instead of complex expression building
	scanInput := &dynamodb.ScanInput{
		TableName:        aws.String(s.tableName),
		Limit:            aws.Int32(int32(limit)),
		FilterExpression: aws.String("EntityType = :entityType"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":entityType": &types.AttributeValueMemberS{Value: "PRODUCT"},
		},
	}

	// Apply cursor if provided
	if cursor != "" {
		exclusiveStartKey, err := decodeCursor(cursor)
		if err != nil {
			return nil, err
		}
		scanInput.ExclusiveStartKey = exclusiveStartKey
	}

	// Execute scan
	result, err := s.db.Scan(ctx, scanInput)
	if err != nil {
		return nil, err
	}

	// Parse products
	products := make([]models.Product, 0, len(result.Items))
	for _, item := range result.Items {
		var dbItem struct {
			Data []byte `dynamodbav:"Data"`
		}
		if err := attributevalue.UnmarshalMap(item, &dbItem); err != nil {
			return nil, err
		}

		var product models.Product
		if err := json.Unmarshal(dbItem.Data, &product); err != nil {
			continue // Skip invalid product data
		}

		// Apply filters in memory (simplified approach)
		if !matchesFilters(product, filters) {
			continue
		}

		products = append(products, product)
	}

	// Get next cursor
	var nextCursor string
	if result.LastEvaluatedKey != nil {
		nextCursor, err = encodeCursor(result.LastEvaluatedKey)
		if err != nil {
			return nil, err
		}
	}

	// Apply sorting (in memory for simplicity)
	sortProducts(products, sortKey, sortOrder)

	return &ProductListResult{
		Products:   products,
		NextCursor: nextCursor,
	}, nil
}

// CountProducts returns the count of products based on filters
func (s *DynamoDBProductService) CountProducts(ctx context.Context, filters map[string]interface{}) (int, error) {
	// Simplified approach - get all products and count after filtering
	result, err := s.ListProducts(ctx, 1000, "", filters, "created_at", "desc")
	if err != nil {
		return 0, err
	}

	return len(result.Products), nil
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
	// First get all products
	productsResult, err := s.ListProducts(ctx, 50, "", filters, sortKey, sortOrder)
	if err != nil {
		return nil, "", err
	}

	// Collect all variants
	allVariants := []models.ProductVariant{}
	for _, product := range productsResult.Products {
		for _, variant := range product.Variants {
			// Add product info to variant
			variant.ProductID = product.ID
			variant.ProductTitle = product.Title
			allVariants = append(allVariants, variant)
		}
	}

	// Apply pagination
	totalVariants := len(allVariants)
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

	// Extract variants for this page
	pageVariants := allVariants[startIndex:endIndex]

	// Calculate next cursor
	var nextCursor string
	if endIndex < totalVariants {
		nextCursor = strconv.Itoa(endIndex)
	}

	return pageVariants, nextCursor, nil
}

// Helper methods

// productKey creates DynamoDB keys for a product
func productKey(productID string) (string, string) {
	return fmt.Sprintf("PRODUCT#%s", productID), fmt.Sprintf("PRODUCT#%s", productID)
}

// matchesFilters checks if a product matches the given filters
func matchesFilters(product models.Product, filters map[string]interface{}) bool {
	if len(filters) == 0 {
		return true
	}

	for key, value := range filters {
		switch key {
		case "status":
			if status, ok := value.(string); ok && product.Status != status {
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

// encodeCursor encodes DynamoDB LastEvaluatedKey to string
func encodeCursor(lastEvaluatedKey map[string]types.AttributeValue) (string, error) {
	// Simple implementation - in real world use base64 encoding
	bytes, err := json.Marshal(lastEvaluatedKey)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// decodeCursor decodes string cursor to DynamoDB ExclusiveStartKey
func decodeCursor(cursor string) (map[string]types.AttributeValue, error) {
	// Simple implementation - in real world use base64 decoding
	var lastEvaluatedKey map[string]types.AttributeValue
	err := json.Unmarshal([]byte(cursor), &lastEvaluatedKey)
	if err != nil {
		return nil, err
	}
	return lastEvaluatedKey, nil
}
