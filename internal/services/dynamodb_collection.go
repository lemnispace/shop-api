package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/internal/utils"
)

// DynamoDBCollectionService is an implementation of CollectionService using DynamoDB
type DynamoDBCollectionService struct {
	db             *dynamodb.Client
	tableName      string
	productService ProductService
}

// NewCollectionService creates a new DynamoDB collection service
func NewCollectionService(db *dynamodb.Client, tableName string, productService ProductService) *DynamoDBCollectionService {
	if db == nil {
		log.Printf("WARNING: DynamoDB client is nil in NewCollectionService")
	}
	if tableName == "" {
		log.Printf("WARNING: Empty table name in NewCollectionService")
		tableName = "ShopAPI" // Default table name
	}
	if productService == nil {
		log.Printf("WARNING: Product service is nil in NewCollectionService")
	}

	log.Printf("Initializing DynamoDB Collection Service with table: %s", tableName)

	return &DynamoDBCollectionService{
		db:             db,
		tableName:      tableName,
		productService: productService,
	}
}

// GetCollection retrieves a collection by ID
func (s *DynamoDBCollectionService) GetCollection(ctx context.Context, id string) (*models.Collection, error) {
	pk, sk := collectionKey(id)

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
		return nil, ErrCollectionNotFound
	}

	var collection models.Collection
	err = attributevalue.UnmarshalMap(result.Item, &collection)
	if err != nil {
		return nil, err
	}

	// Load products for this collection
	products, _, err := s.ListCollectionProducts(ctx, id, 100, "")
	if err != nil {
		return nil, err
	}
	collection.Products = products

	return &collection, nil
}

// Helper function to generate a unique ID
func generateID() string {
	return fmt.Sprintf("col_%d", time.Now().UnixNano())
}

// CreateCollection creates a new collection
func (s *DynamoDBCollectionService) CreateCollection(ctx context.Context, collection *models.Collection) error {
	if s.db == nil {
		log.Printf("ERROR: DynamoDB client is nil in CreateCollection")
		return fmt.Errorf("dynamoDB client not initialized")
	}

	// Generate ID if not provided
	if collection.ID == "" {
		collection.ID = generateID()
		log.Printf("Generated new collection ID: %s", collection.ID)
	}

	// Set timestamps if not already set
	now := time.Now().UTC()
	if collection.CreatedAt.IsZero() {
		collection.CreatedAt = now
	}
	collection.UpdatedAt = now

	log.Printf("Creating collection with ID: %s", collection.ID)

	// Convert collection to DynamoDB item
	item, err := attributevalue.MarshalMap(collection)
	if err != nil {
		utils.ErrorLog("Failed to marshal collection: %v", err)
		return err
	}

	// Add key attributes
	pk, sk := CollectionKey(collection.ID)
	item["PK"] = &types.AttributeValueMemberS{Value: pk}
	item["SK"] = &types.AttributeValueMemberS{Value: sk}
	item["EntityType"] = &types.AttributeValueMemberS{Value: EntityCollection}

	// Put item in DynamoDB
	putItemInput := &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	}

	log.Printf("Putting item in DynamoDB table: %s", s.tableName)

	_, err = s.db.PutItem(ctx, putItemInput)
	if err != nil {
		log.Printf("Error putting item in DynamoDB: %v", err)
		return fmt.Errorf("failed to create collection: %w", err)
	}

	log.Printf("Successfully created collection %s in DynamoDB", collection.ID)

	// Add product relationships if any
	if len(collection.Products) > 0 {
		for _, product := range collection.Products {
			err := s.AddProductToCollection(ctx, collection.ID, product.ID)
			if err != nil {
				log.Printf("Error adding product %s to collection: %v", product.ID, err)
				// Continue even if some products fail
				continue
			}
		}
	}

	return nil
}

// UpdateCollection updates an existing collection
func (s *DynamoDBCollectionService) UpdateCollection(ctx context.Context, collection *models.Collection) error {
	// Check if collection exists
	existingCollection, err := s.GetCollection(ctx, collection.ID)
	if err != nil {
		return err
	}

	// Preserve creation timestamp
	collection.CreatedAt = existingCollection.CreatedAt
	collection.UpdatedAt = time.Now()

	// Make a copy without products to store in DynamoDB
	collectionToStore := *collection
	collectionToStore.Products = nil

	pk, sk := collectionKey(collection.ID)
	item, err := attributevalue.MarshalMap(collectionToStore)
	if err != nil {
		return err
	}

	// Add PK and SK
	item["PK"] = &types.AttributeValueMemberS{Value: pk}
	item["SK"] = &types.AttributeValueMemberS{Value: sk}

	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	if err != nil {
		return err
	}

	return nil
}

// DeleteCollection deletes a collection
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

	// Check if collection exists first
	exists, err := s.collectionExists(ctx, id)
	if err != nil {
		utils.ErrorLog("Error checking if collection exists: %v", err)
		return fmt.Errorf("error checking collection existence: %w", err)
	}

	if !exists {
		utils.ErrorLog("Collection not found with ID: %s", id)
		return ErrCollectionNotFound
	}

	// Delete collection metadata
	pk, sk := CollectionKey(id)
	utils.DebugLog("Using collection keys - PK: %s, SK: %s", pk, sk)

	// Delete collection
	utils.DebugLog("Deleting collection from DynamoDB")
	_, err = s.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})

	if err != nil {
		utils.ErrorLog("Failed to delete collection from DynamoDB: %v", err)
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	// Delete products-collection relationships
	utils.DebugLog("Deleting collection-product relationships for collection: %s", id)
	err = s.deleteCollectionProducts(ctx, id)
	if err != nil {
		utils.ErrorLog("Failed to delete collection-product relationships: %v", err)
		return fmt.Errorf("failed to delete collection products: %w", err)
	}

	utils.DebugLog("Successfully deleted collection with ID: %s", id)
	return nil
}

// Helper method to check if a collection exists
func (s *DynamoDBCollectionService) collectionExists(ctx context.Context, id string) (bool, error) {
	utils.DebugLog("Checking if collection exists: %s", id)

	if id == "" {
		return false, fmt.Errorf("collection ID cannot be empty")
	}

	pk, sk := CollectionKey(id)

	result, err := s.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
		ProjectionExpression: aws.String("PK"),
	})

	if err != nil {
		return false, err
	}

	return len(result.Item) > 0, nil
}

// ListCollections retrieves a list of collections with pagination
func (s *DynamoDBCollectionService) ListCollections(ctx context.Context, limit int, cursor string, filters map[string]interface{}, sortKey, sortOrder string) (*CollectionListResult, error) {
	utils.DebugLog("Listing collections with limit: %d, cursor: %s, filters: %v, sort: %s %s",
		limit, cursor, filters, sortKey, sortOrder)

	if s.db == nil {
		utils.ErrorLog("DynamoDB client is nil in ListCollections")
		return nil, fmt.Errorf("dynamoDB client not initialized")
	}

	if limit <= 0 {
		limit = 20 // Default limit
	}

	// Scan parameters for table scan approach
	scanParams := &dynamodb.ScanInput{
		TableName:        aws.String(s.tableName),
		Limit:            aws.Int32(int32(limit)),
		FilterExpression: aws.String("begins_with(PK, :pk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#", EntityCollection)},
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
		scanParams.ExclusiveStartKey = exclusiveStartKey
	}

	utils.DebugLog("Executing DynamoDB scan: %+v", scanParams)

	// Execute scan
	result, err := s.db.Scan(ctx, scanParams)
	if err != nil {
		utils.ErrorLog("DynamoDB scan failed: %v", err)
		return nil, fmt.Errorf("failed to query collections: %w", err)
	}

	utils.DebugLog("Scan returned %d items", len(result.Items))

	// Unmarshal results
	var collections []models.Collection
	for _, item := range result.Items {
		var collection models.Collection
		if err := attributevalue.UnmarshalMap(item, &collection); err != nil {
			utils.ErrorLog("Failed to unmarshal collection: %v", err)
			continue
		}

		// Extract ID from PK if not set directly
		if collection.ID == "" && item["PK"] != nil {
			if pk, ok := item["PK"].(*types.AttributeValueMemberS); ok {
				collection.ID = utils.ExtractIDFromPK(pk.Value)
			}
		}

		utils.DebugLog("Found collection: %s - %s", collection.ID, collection.Title)

		// Load products for this collection
		products, _, err := s.ListCollectionProducts(ctx, collection.ID, 100, "")
		if err != nil {
			utils.ErrorLog("Failed to load products for collection %s: %v", collection.ID, err)
			// Continue with empty products list
			collection.Products = []models.Product{}
		} else {
			collection.Products = products
		}

		collections = append(collections, collection)
	}

	// Apply custom filters if needed
	if len(filters) > 0 {
		utils.DebugLog("Applying custom filters: %v", filters)
		var filteredCollections []models.Collection
		for _, collection := range collections {
			if utils.MatchesCollectionFilters(collection, filters) {
				filteredCollections = append(filteredCollections, collection)
			}
		}
		collections = filteredCollections
	}

	// Sort collections if needed
	if sortKey != "" {
		utils.DebugLog("Sorting collections by %s %s", sortKey, sortOrder)
		utils.SortCollections(collections, sortKey, sortOrder)
	}

	// Get next page cursor
	var nextCursor string
	if len(result.LastEvaluatedKey) > 0 {
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

// CountCollections counts the total number of collections that match the given filters
func (s *DynamoDBCollectionService) CountCollections(ctx context.Context, filters map[string]interface{}) (int, error) {
	if s.db == nil {
		utils.ErrorLog("DynamoDB client is nil in CountCollections")
		return 0, fmt.Errorf("dynamoDB client not initialized")
	}

	// Scan parameters
	scanParams := &dynamodb.ScanInput{
		TableName:        aws.String(s.tableName),
		FilterExpression: aws.String("begins_with(PK, :pk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#", EntityCollection)},
		},
		Select: types.SelectCount,
	}

	utils.DebugLog("Executing DynamoDB count scan: %+v", scanParams)

	// Execute scan
	result, err := s.db.Scan(ctx, scanParams)
	if err != nil {
		utils.ErrorLog("DynamoDB count scan failed: %v", err)
		return 0, fmt.Errorf("failed to count collections: %w", err)
	}

	utils.DebugLog("Count scan returned %d items", result.Count)

	return int(result.Count), nil
}

// AddProductToCollection adds a product to a collection
func (s *DynamoDBCollectionService) AddProductToCollection(ctx context.Context, collectionID, productID string) error {
	utils.DebugLog("Adding product %s to collection %s", productID, collectionID)

	if s.db == nil {
		utils.ErrorLog("DynamoDB client is nil in AddProductToCollection")
		return fmt.Errorf("dynamoDB client not initialized")
	}

	// Verify collection exists - this could hang if the GetCollection method has issues
	collection, err := s.GetCollection(ctx, collectionID)
	if err != nil {
		utils.ErrorLog("Failed to get collection %s: %v", collectionID, err)
		return err
	}
	utils.DebugLog("Found collection: %s - %s", collection.ID, collection.Title)

	// Verify product exists
	product, err := s.productService.GetProduct(ctx, productID)
	if err != nil {
		utils.ErrorLog("Failed to get product %s: %v", productID, err)
		return err
	}
	utils.DebugLog("Found product: %s - %s", product.ID, product.Title)

	// Create the collection-product relationship
	pk := fmt.Sprintf("%s#%s", EntityCollection, collectionID)
	sk := fmt.Sprintf("%s#%s", EntityProduct, productID)
	utils.DebugLog("Creating collection-product relationship with PK: %s, SK: %s", pk, sk)

	// Create the relationship item for DynamoDB
	relationshipItem := map[string]types.AttributeValue{
		"PK":           &types.AttributeValueMemberS{Value: pk},
		"SK":           &types.AttributeValueMemberS{Value: sk},
		"EntityType":   &types.AttributeValueMemberS{Value: EntityCollectionProductRel},
		"CollectionID": &types.AttributeValueMemberS{Value: collectionID},
		"ProductID":    &types.AttributeValueMemberS{Value: productID},
		"CreatedAt":    &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
	}

	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      relationshipItem,
	})
	if err != nil {
		utils.ErrorLog("Failed to put collection-product relationship: %v", err)
		return fmt.Errorf("error adding product to collection: %w", err)
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

	// Create a context with timeout to prevent hanging
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Verify collection exists
	_, err := s.GetCollection(ctxWithTimeout, collectionID)
	if err != nil {
		utils.ErrorLog("Failed to verify collection existence: %v", err)
		return err
	}

	// Delete the collection-product relationship
	pk := fmt.Sprintf("%s#%s", EntityCollection, collectionID)
	sk := fmt.Sprintf("%s#%s", EntityProduct, productID)
	utils.DebugLog("Deleting collection-product relationship with PK: %s, SK: %s", pk, sk)

	// Delete the relationship
	_, err = s.db.DeleteItem(ctxWithTimeout, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})

	if err != nil {
		utils.ErrorLog("Failed to delete collection-product relationship: %v", err)
		return fmt.Errorf("error removing product from collection: %w", err)
	}

	utils.DebugLog("Successfully removed product %s from collection %s", productID, collectionID)
	return nil
}

// ListCollectionProducts retrieves all products in a collection with pagination
func (s *DynamoDBCollectionService) ListCollectionProducts(ctx context.Context, collectionID string, limit int, cursor string) ([]models.Product, string, error) {
	utils.DebugLog("Listing products for collection %s with limit: %d, cursor: %s",
		collectionID, limit, cursor)

	if s.db == nil {
		utils.ErrorLog("DynamoDB client is nil in ListCollectionProducts")
		return nil, "", fmt.Errorf("dynamoDB client not initialized")
	}

	if limit <= 0 {
		limit = 20 // Default limit
	}

	// First, verify the collection exists
	exists, err := s.collectionExists(ctx, collectionID)
	if err != nil {
		utils.ErrorLog("Error checking if collection exists: %v", err)
		return nil, "", fmt.Errorf("error checking collection: %w", err)
	}

	if !exists {
		utils.ErrorLog("Collection not found: %s", collectionID)
		return nil, "", ErrCollectionNotFound
	}

	// Use absolute direct key patterns for reliability
	collectionPrefix := fmt.Sprintf("%s#%s", EntityCollection, collectionID)
	utils.DebugLog("Using collection prefix: %s", collectionPrefix)

	// Try a direct GetItem for the collection first to double-check it exists
	collectionItem, err := s.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: collectionPrefix},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("METADATA#%s", collectionID)},
		},
	})

	// Log the actual outcome instead of every possible path
	if err != nil {
		utils.DebugLog("Using alternate collection lookup methods due to error: %v", err)
	} else if collectionItem.Item == nil {
		utils.DebugLog("Using alternate collection lookup methods: direct lookup returned no items")
	}

	// Try both scanning and querying to be extra safe
	// Method 1: Scan-based approach
	utils.DebugLog("Trying scan-based approach for collection products")
	scanInput := &dynamodb.ScanInput{
		TableName:        aws.String(s.tableName),
		Limit:            aws.Int32(int32(limit)),
		FilterExpression: aws.String("begins_with(PK, :pk) AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: collectionPrefix},
			":sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#", EntityProduct)},
		},
	}

	utils.DebugLog("Executing DynamoDB scan: %+v", scanInput)

	scanResult, err := s.db.Scan(ctx, scanInput)
	if err != nil {
		utils.ErrorLog("DynamoDB scan failed: %v", err)
		// Don't return, try the query approach instead
	} else {
		utils.DebugLog("Scan returned %d items", len(scanResult.Items))
		// Only log the summary, not every item
	}

	// Method 2: Query-based approach as a backup
	utils.DebugLog("Trying query-based approach as backup")
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: collectionPrefix},
			":sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#", EntityProduct)},
		},
		Limit: aws.Int32(int32(limit)),
	}

	utils.DebugLog("Executing DynamoDB query: %+v", queryInput)

	queryResult, err := s.db.Query(ctx, queryInput)
	if err != nil {
		utils.ErrorLog("DynamoDB query failed: %v", err)
		// If both approaches failed, we'll have to return an empty result
		if scanResult == nil || len(scanResult.Items) == 0 {
			utils.DebugLog("Both scan and query approaches failed or returned no items")
			return []models.Product{}, "", nil
		}
	} else {
		utils.DebugLog("Query returned %d items", len(queryResult.Items))
		// Only log the summary, not every item
	}

	// Use the result with more items (or any non-empty result)
	var result *dynamodb.ScanOutput
	var resultItems []map[string]types.AttributeValue

	if scanResult != nil && len(scanResult.Items) > 0 {
		utils.DebugLog("Using scan results with %d items", len(scanResult.Items))
		resultItems = scanResult.Items
		result = scanResult
	} else if queryResult != nil && len(queryResult.Items) > 0 {
		utils.DebugLog("Using query results with %d items", len(queryResult.Items))
		resultItems = queryResult.Items
		result = &dynamodb.ScanOutput{
			Items:            queryResult.Items,
			LastEvaluatedKey: queryResult.LastEvaluatedKey,
			Count:            queryResult.Count,
		}
	} else {
		utils.DebugLog("No products found for collection %s", collectionID)
		return []models.Product{}, "", nil
	}

	// Extract product IDs from the relationships
	var productIDs []string

	for _, item := range resultItems {
		// First try finding the ProductID field directly
		if productID, ok := item["ProductID"]; ok {
			if productIDStr, ok := productID.(*types.AttributeValueMemberS); ok {
				utils.DebugLog("Found product ID from ProductID field: %s", productIDStr.Value)
				productIDs = append(productIDs, productIDStr.Value)
				continue
			}
		}

		// Fallback to parsing from SK if ProductID isn't available
		sk, ok := item["SK"].(*types.AttributeValueMemberS)
		if !ok {
			utils.ErrorLog("SK is not a string in result item")
			continue
		}

		// Extract product ID from SK (PRODUCT#<id>)
		parts := strings.Split(sk.Value, "#")
		if len(parts) < 2 {
			utils.ErrorLog("Invalid SK format: %s", sk.Value)
			continue
		}

		productID := parts[1]
		utils.DebugLog("Found product ID from SK: %s", productID)
		productIDs = append(productIDs, productID)
	}

	utils.DebugLog("Extracted %d product IDs from the collection", len(productIDs))

	// Get the products by their IDs
	var products []models.Product
	for _, productID := range productIDs {
		utils.DebugLog("Fetching product %s", productID)
		product, err := s.productService.GetProduct(ctx, productID)
		if err != nil {
			utils.ErrorLog("Failed to fetch product %s: %v", productID, err)
			// Continue with other products
			continue
		}
		utils.DebugLog("Retrieved product: %s - %s", product.ID, product.Title)
		products = append(products, *product)
	}

	// Get next page cursor
	var nextCursor string
	if len(result.LastEvaluatedKey) > 0 {
		utils.DebugLog("Generating next cursor from LastEvaluatedKey")
		nextCursor, err = utils.EncodeCursor(result.LastEvaluatedKey)
		if err != nil {
			utils.ErrorLog("Failed to encode cursor: %v", err)
			// Continue without cursor
		}
	}

	utils.DebugLog("Returning %d products with nextCursor: %s", len(products), nextCursor)
	return products, nextCursor, nil
}

// deleteCollectionProducts removes all product associations for a collection
func (s *DynamoDBCollectionService) deleteCollectionProducts(ctx context.Context, collectionID string) error {
	// In a real implementation, this would use a batch delete or transaction
	// For now, we'll use a query + delete for each item
	pk := fmt.Sprintf("%s#%s", EntityCollection, collectionID)

	// Query all products for this collection
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: pk},
			":sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("%s#", EntityProduct)},
		},
	}

	result, err := s.db.Query(ctx, queryInput)
	if err != nil {
		return err
	}

	// Delete each item
	for _, item := range result.Items {
		pk := item["PK"].(*types.AttributeValueMemberS).Value
		sk := item["SK"].(*types.AttributeValueMemberS).Value

		_, err := s.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
			TableName: aws.String(s.tableName),
			Key: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: pk},
				"SK": &types.AttributeValueMemberS{Value: sk},
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// collectionKey creates keys for a collection (using the service function)
func collectionKey(collectionID string) (string, string) {
	return CollectionKey(collectionID)
}
