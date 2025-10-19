package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/internal/utils"
)

// GetCollection retrieves a collection by ID with its products
func (s *DynamoDBCollectionService) GetCollection(ctx context.Context, id string) (*models.Collection, error) {
	utils.DebugLog("Getting collection with ID: %s", id)

	if s.db == nil {
		utils.ErrorLog("DynamoDB client is nil in GetCollection")
		return nil, fmt.Errorf("dynamoDB client not initialized")
	}

	if id == "" {
		utils.ErrorLog("Empty collection ID provided to GetCollection")
		return nil, fmt.Errorf("collection ID cannot be empty")
	}

	// Get collection metadata
	pk, sk := CollectionKey(id)
	utils.DebugLog("Using collection keys - PK: %s, SK: %s", pk, sk)

	result, err := s.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		utils.ErrorLog("Failed to get collection from DynamoDB: %v", err)
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	if result.Item == nil || len(result.Item) == 0 {
		utils.ErrorLog("Collection not found with ID: %s", id)
		return nil, ErrCollectionNotFound
	}

	// Unmarshal collection data
	var collection models.Collection
	dataAttr, ok := result.Item["Data"]
	if ok {
		dataBytes, ok := dataAttr.(*types.AttributeValueMemberB)
		if ok {
			if err := json.Unmarshal(dataBytes.Value, &collection); err != nil {
				utils.ErrorLog("Failed to unmarshal collection data: %v", err)
				return nil, fmt.Errorf("failed to unmarshal collection: %w", err)
			}
		}
	} else {
		// Try direct unmarshal if no Data field
		if err := attributevalue.UnmarshalMap(result.Item, &collection); err != nil {
			utils.ErrorLog("Failed to unmarshal collection: %v", err)
			return nil, fmt.Errorf("failed to unmarshal collection: %w", err)
		}
	}

	// Ensure ID is set
	if collection.ID == "" {
		collection.ID = id
	}

	// Get products in the collection
	products, err := s.getCollectionProductsInternal(ctx, id)
	if err != nil {
		utils.ErrorLog("Failed to get collection products: %v", err)
		// Continue even if we can't get products
		products = []models.Product{}
	}

	collection.Products = products

	utils.DebugLog("Successfully retrieved collection: %s with %d products", collection.ID, len(collection.Products))
	return &collection, nil
}

// CreateCollection creates a new collection
func (s *DynamoDBCollectionService) CreateCollection(ctx context.Context, collection *models.Collection) error {
	utils.DebugLog("Creating collection: %s", collection.Title)

	if s.db == nil {
		utils.ErrorLog("DynamoDB client is nil in CreateCollection")
		return fmt.Errorf("dynamoDB client not initialized")
	}

	// Generate ID if not provided
	if collection.ID == "" {
		collection.ID = fmt.Sprintf("col_%d", time.Now().UnixNano())
		utils.DebugLog("Generated collection ID: %s", collection.ID)
	}

	// Set timestamps
	now := time.Now()
	collection.CreatedAt = now
	collection.UpdatedAt = now

	// Marshal collection data
	data, err := json.Marshal(collection)
	if err != nil {
		utils.ErrorLog("Failed to marshal collection: %v", err)
		return fmt.Errorf("failed to marshal collection: %w", err)
	}

	// Get keys
	pk, sk := CollectionKey(collection.ID)
	utils.DebugLog("Using collection keys - PK: %s, SK: %s", pk, sk)

	// Create item
	item := map[string]types.AttributeValue{
		"PK":         &types.AttributeValueMemberS{Value: pk},
		"SK":         &types.AttributeValueMemberS{Value: sk},
		"EntityType": &types.AttributeValueMemberS{Value: EntityCollection},
		"Data":       &types.AttributeValueMemberB{Value: data},
		"CreatedAt":  &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		"UpdatedAt":  &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
	}

	// Store collection in DynamoDB
	utils.DebugLog("Storing collection in DynamoDB")
	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})

	if err != nil {
		utils.ErrorLog("Failed to store collection in DynamoDB: %v", err)
		return fmt.Errorf("failed to store collection: %w", err)
	}

	utils.DebugLog("Successfully created collection with ID: %s", collection.ID)
	return nil
}

// UpdateCollection updates an existing collection
func (s *DynamoDBCollectionService) UpdateCollection(ctx context.Context, collection *models.Collection) error {
	utils.DebugLog("Updating collection with ID: %s", collection.ID)

	if s.db == nil {
		utils.ErrorLog("DynamoDB client is nil in UpdateCollection")
		return fmt.Errorf("dynamoDB client not initialized")
	}

	if collection.ID == "" {
		utils.ErrorLog("Empty collection ID provided to UpdateCollection")
		return fmt.Errorf("collection ID cannot be empty")
	}

	// Check if collection exists
	existingCollection, err := s.GetCollection(ctx, collection.ID)
	if err != nil {
		utils.ErrorLog("Failed to find collection to update: %v", err)
		return fmt.Errorf("cannot update collection: %w", err)
	}

	// Preserve timestamps
	collection.CreatedAt = existingCollection.CreatedAt
	collection.UpdatedAt = time.Now()

	// Marshal collection data
	data, err := json.Marshal(collection)
	if err != nil {
		utils.ErrorLog("Failed to marshal collection: %v", err)
		return fmt.Errorf("failed to marshal collection: %w", err)
	}

	// Get keys
	pk, sk := CollectionKey(collection.ID)
	utils.DebugLog("Using collection keys - PK: %s, SK: %s", pk, sk)

	// Create item
	item := map[string]types.AttributeValue{
		"PK":         &types.AttributeValueMemberS{Value: pk},
		"SK":         &types.AttributeValueMemberS{Value: sk},
		"EntityType": &types.AttributeValueMemberS{Value: EntityCollection},
		"Data":       &types.AttributeValueMemberB{Value: data},
		"CreatedAt":  &types.AttributeValueMemberS{Value: collection.CreatedAt.Format(time.RFC3339)},
		"UpdatedAt":  &types.AttributeValueMemberS{Value: collection.UpdatedAt.Format(time.RFC3339)},
	}

	// Store updated collection in DynamoDB
	utils.DebugLog("Storing updated collection in DynamoDB")
	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})

	if err != nil {
		utils.ErrorLog("Failed to update collection in DynamoDB: %v", err)
		return fmt.Errorf("failed to update collection: %w", err)
	}

	utils.DebugLog("Successfully updated collection with ID: %s", collection.ID)
	return nil
}

// DeleteCollection deletes a collection and all its product relationships
func (s *DynamoDBCollectionService) DeleteCollection(ctx context.Context, id string) error {
	utils.DebugLog("Deleting collection with ID: %s", id)

	if s.db == nil {
		utils.ErrorLog("DynamoDB client is nil in DeleteCollection")
		return fmt.Errorf("dynamoDB client not initialized")
	}

	if id == "" {
		utils.ErrorLog("Empty collection ID provided to DeleteCollection")
		return fmt.Errorf("collection ID cannot be empty")
	}

	// Check if collection exists
	_, err := s.GetCollection(ctx, id)
	if err != nil {
		if err == ErrCollectionNotFound {
			return err
		}
		utils.ErrorLog("Error checking if collection exists: %v", err)
		return fmt.Errorf("error checking collection existence: %w", err)
	}

	// Delete all product relationships first
	pk := fmt.Sprintf("%s#%s", EntityCollection, id)
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk_prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":        &types.AttributeValueMemberS{Value: pk},
			":sk_prefix": &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#", EntityProduct)},
		},
	}

	result, err := s.db.Query(ctx, queryInput)
	if err != nil {
		utils.ErrorLog("Failed to query collection products: %v", err)
		return fmt.Errorf("failed to query collection products: %w", err)
	}

	// Delete each product relationship
	for _, item := range result.Items {
		skAttr, ok := item["SK"]
		if !ok {
			continue
		}
		sk, ok := skAttr.(*types.AttributeValueMemberS)
		if !ok {
			continue
		}

		_, err = s.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
			TableName: aws.String(s.tableName),
			Key: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: pk},
				"SK": sk,
			},
		})
		if err != nil {
			utils.ErrorLog("Failed to delete product relationship: %v", err)
			// Continue with other deletions
		}
	}

	// Delete collection metadata
	collPK, collSK := CollectionKey(id)
	_, err = s.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: collPK},
			"SK": &types.AttributeValueMemberS{Value: collSK},
		},
	})
	if err != nil {
		utils.ErrorLog("Failed to delete collection from DynamoDB: %v", err)
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	utils.DebugLog("Successfully deleted collection with ID: %s", id)
	return nil
}

// ListCollections lists collections with pagination
func (s *DynamoDBCollectionService) ListCollections(ctx context.Context, limit int, cursor string, filters map[string]interface{}, sortKey, sortOrder string) (*CollectionListResult, error) {
	utils.DebugLog("Listing collections with limit: %d, cursor: %s", limit, cursor)

	if s.db == nil {
		utils.ErrorLog("DynamoDB client is nil in ListCollections")
		return nil, fmt.Errorf("dynamoDB client not initialized")
	}

	if limit <= 0 {
		limit = 20 // Default limit
	}

	// Use a scan with filter to find collections
	scanInput := &dynamodb.ScanInput{
		TableName:        aws.String(s.tableName),
		Limit:            aws.Int32(int32(limit)),
		FilterExpression: aws.String("begins_with(PK, :pk) AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#", EntityCollection)},
			":sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#", EntityCollection)},
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

	utils.DebugLog("Executing DynamoDB scan")

	// Execute scan
	result, err := s.db.Scan(ctx, scanInput)
	if err != nil {
		utils.ErrorLog("DynamoDB scan failed: %v", err)
		return nil, fmt.Errorf("failed to query collections: %w", err)
	}

	utils.DebugLog("Scan returned %d items", len(result.Items))

	// Unmarshal results
	var collections []models.Collection
	for _, item := range result.Items {
		var collection models.Collection

		// Try to get the collection from the Data field
		dataAttr, ok := item["Data"]
		if ok {
			dataBytes, ok := dataAttr.(*types.AttributeValueMemberB)
			if ok {
				if err := json.Unmarshal(dataBytes.Value, &collection); err != nil {
					utils.ErrorLog("Failed to unmarshal collection data: %v", err)
					continue
				}
			}
		} else {
			// If no Data field, try direct unmarshal
			if err := attributevalue.UnmarshalMap(item, &collection); err != nil {
				utils.ErrorLog("Failed to unmarshal collection: %v", err)
				continue
			}
		}

		// Extract ID from PK if not set directly
		if collection.ID == "" && item["PK"] != nil {
			if pk, ok := item["PK"].(*types.AttributeValueMemberS); ok {
				parts := strings.Split(pk.Value, "#")
				if len(parts) > 1 {
					collection.ID = parts[1]
				}
			}
		}

		utils.DebugLog("Found collection: %s - %s", collection.ID, collection.Title)
		collections = append(collections, collection)
	}

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

	utils.DebugLog("Returning %d collections with nextCursor: %s", len(collections), nextCursor)

	return &CollectionListResult{
		Collections: collections,
		NextCursor:  nextCursor,
	}, nil
}

// CountCollections returns the count of collections
func (s *DynamoDBCollectionService) CountCollections(ctx context.Context, filters map[string]interface{}) (int, error) {
	utils.DebugLog("Counting collections with filters: %v", filters)

	if s.db == nil {
		utils.ErrorLog("DynamoDB client is nil in CountCollections")
		return 0, fmt.Errorf("dynamoDB client not initialized")
	}

	// Use a scan operation to count collections
	scanInput := &dynamodb.ScanInput{
		TableName:        aws.String(s.tableName),
		FilterExpression: aws.String("begins_with(PK, :pk) AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#", EntityCollection)},
			":sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#", EntityCollection)},
		},
		Select: types.SelectCount,
	}

	utils.DebugLog("Executing DynamoDB count scan")

	// Execute scan
	result, err := s.db.Scan(ctx, scanInput)
	if err != nil {
		utils.ErrorLog("DynamoDB count scan failed: %v", err)
		return 0, fmt.Errorf("failed to count collections: %w", err)
	}

	count := int(result.Count)
	utils.DebugLog("Count scan returned %d items", count)

	return count, nil
}

// AddProductToCollection adds a product to a collection
func (s *DynamoDBCollectionService) AddProductToCollection(ctx context.Context, collectionID, productID string) error {
	utils.DebugLog("Adding product %s to collection %s", productID, collectionID)

	if s.db == nil {
		utils.ErrorLog("DynamoDB client is nil in AddProductToCollection")
		return fmt.Errorf("dynamoDB client not initialized")
	}

	// Check if collection exists
	_, err := s.GetCollection(ctx, collectionID)
	if err != nil {
		return err
	}

	// Check if product exists
	if s.productService != nil {
		_, err = s.productService.GetProduct(ctx, productID)
		if err != nil {
			return fmt.Errorf("product not found: %w", err)
		}
	}

	// Create collection-product relationship
	pk, sk := CollectionProductKey(collectionID, productID)
	now := time.Now()

	item := map[string]types.AttributeValue{
		"PK":         &types.AttributeValueMemberS{Value: pk},
		"SK":         &types.AttributeValueMemberS{Value: sk},
		"EntityType": &types.AttributeValueMemberS{Value: EntityCollectionProductRel},
		"AddedAt":    &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
	}

	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})

	if err != nil {
		utils.ErrorLog("Failed to add product to collection: %v", err)
		return fmt.Errorf("failed to add product to collection: %w", err)
	}

	utils.DebugLog("Successfully added product %s to collection %s", productID, collectionID)
	return nil
}

// RemoveProductFromCollection removes a product from a collection
func (s *DynamoDBCollectionService) RemoveProductFromCollection(ctx context.Context, collectionID, productID string) error {
	utils.DebugLog("Removing product %s from collection %s", productID, collectionID)

	if s.db == nil {
		utils.ErrorLog("DynamoDB client is nil in RemoveProductFromCollection")
		return fmt.Errorf("dynamoDB client not initialized")
	}

	// Delete collection-product relationship
	pk, sk := CollectionProductKey(collectionID, productID)

	_, err := s.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})

	if err != nil {
		utils.ErrorLog("Failed to remove product from collection: %v", err)
		return fmt.Errorf("failed to remove product from collection: %w", err)
	}

	utils.DebugLog("Successfully removed product %s from collection %s", productID, collectionID)
	return nil
}

// ListCollectionProducts lists products in a collection with pagination
func (s *DynamoDBCollectionService) ListCollectionProducts(ctx context.Context, collectionID string, limit int, cursor string) ([]models.Product, string, error) {
	utils.DebugLog("Listing products in collection %s with limit: %d", collectionID, limit)

	if s.db == nil {
		utils.ErrorLog("DynamoDB client is nil in ListCollectionProducts")
		return nil, "", fmt.Errorf("dynamoDB client not initialized")
	}

	products, err := s.getCollectionProductsInternal(ctx, collectionID)
	if err != nil {
		return nil, "", err
	}

	// Apply simple pagination (in-memory)
	// In production, this should use DynamoDB pagination
	if limit <= 0 {
		limit = 20
	}

	start := 0
	if cursor != "" {
		// Simple cursor parsing (offset-based)
		fmt.Sscanf(cursor, "offset_%d", &start)
	}

	end := start + limit
	if end > len(products) {
		end = len(products)
	}

	var paginatedProducts []models.Product
	if start < len(products) {
		paginatedProducts = products[start:end]
	} else {
		paginatedProducts = []models.Product{}
	}

	var nextCursor string
	if end < len(products) {
		nextCursor = fmt.Sprintf("offset_%d", end)
	}

	utils.DebugLog("Returning %d products with nextCursor: %s", len(paginatedProducts), nextCursor)
	return paginatedProducts, nextCursor, nil
}

// CountCollectionProducts returns the count of products in a collection efficiently
func (s *DynamoDBCollectionService) CountCollectionProducts(ctx context.Context, collectionID string) (int, error) {
	utils.DebugLog("Counting products for collection: %s", collectionID)

	if s.db == nil {
		utils.ErrorLog("DynamoDB client is nil in CountCollectionProducts")
		return 0, fmt.Errorf("dynamoDB client not initialized")
	}

	// Query for all product relationships with Select COUNT
	pk := fmt.Sprintf("%s#%s", EntityCollection, collectionID)
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk_prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":        &types.AttributeValueMemberS{Value: pk},
			":sk_prefix": &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#", EntityProduct)},
		},
		Select: types.SelectCount, // Only count, don't fetch data
	}

	result, err := s.db.Query(ctx, queryInput)
	if err != nil {
		utils.ErrorLog("Failed to count collection products: %v", err)
		return 0, fmt.Errorf("failed to count collection products: %w", err)
	}

	count := int(result.Count)
	utils.DebugLog("Collection %s has %d products", collectionID, count)
	return count, nil
}

// getCollectionProductsInternal retrieves all products for a collection (internal helper)
func (s *DynamoDBCollectionService) getCollectionProductsInternal(ctx context.Context, collectionID string) ([]models.Product, error) {
	// Query for all product relationships
	pk := fmt.Sprintf("%s#%s", EntityCollection, collectionID)
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk_prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":        &types.AttributeValueMemberS{Value: pk},
			":sk_prefix": &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#", EntityProduct)},
		},
	}

	result, err := s.db.Query(ctx, queryInput)
	if err != nil {
		utils.ErrorLog("Failed to query collection products: %v", err)
		return nil, fmt.Errorf("failed to query collection products: %w", err)
	}

	// Extract product IDs and fetch full product data
	var products []models.Product
	for _, item := range result.Items {
		skAttr, ok := item["SK"]
		if !ok {
			continue
		}
		sk, ok := skAttr.(*types.AttributeValueMemberS)
		if !ok {
			continue
		}

		// Extract product ID from SK
		parts := strings.Split(sk.Value, "#")
		if len(parts) < 2 {
			continue
		}
		productID := parts[1]

		// Fetch full product data if product service is available
		if s.productService != nil {
			product, err := s.productService.GetProduct(ctx, productID)
			if err != nil {
				utils.ErrorLog("Failed to get product %s: %v", productID, err)
				continue
			}
			products = append(products, *product)
		}
	}

	return products, nil
}
