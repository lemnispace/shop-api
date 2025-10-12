package services

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/lemnispace/shop-api/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDynamoDB(t *testing.T) (*dynamodb.Client, string) {
	endpoint := os.Getenv("DYNAMODB_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8000"
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               endpoint,
			HostnameImmutable: true,
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("minioadmin", "minioadmin", "")),
	)
	require.NoError(t, err)

	client := dynamodb.NewFromConfig(cfg)

	// Create test table
	tableName := "ShopAPI-Test-" + time.Now().Format("20060102150405")

	_, err = client.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("PK"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("SK"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("GSI1PK"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("GSI1SK"), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("PK"), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String("SK"), KeyType: types.KeyTypeRange},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("GSI1"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("GSI1PK"), KeyType: types.KeyTypeHash},
					{AttributeName: aws.String("GSI1SK"), KeyType: types.KeyTypeRange},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeAll,
				},
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(5),
					WriteCapacityUnits: aws.Int64(5),
				},
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
	})
	require.NoError(t, err)

	// Wait for table to be active
	time.Sleep(2 * time.Second)

	return client, tableName
}

func cleanupTestTable(t *testing.T, client *dynamodb.Client, tableName string) {
	_, err := client.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		t.Logf("Failed to delete test table: %v", err)
	}
}

func TestCollectionService_CreateCollection(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	collectionService := NewCollectionService(client, tableName, productService)

	collection := &models.Collection{
		Title:       "Test Collection",
		Description: "A test collection",
	}

	err := collectionService.CreateCollection(context.Background(), collection)
	require.NoError(t, err)
	assert.NotEmpty(t, collection.ID)
	assert.NotZero(t, collection.CreatedAt)
	assert.NotZero(t, collection.UpdatedAt)
}

func TestCollectionService_GetCollection(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	collectionService := NewCollectionService(client, tableName, productService)

	// Create a collection
	collection := &models.Collection{
		Title:       "Test Collection",
		Description: "A test collection",
	}

	err := collectionService.CreateCollection(context.Background(), collection)
	require.NoError(t, err)

	// Get the collection
	retrieved, err := collectionService.GetCollection(context.Background(), collection.ID)
	require.NoError(t, err)
	assert.Equal(t, collection.ID, retrieved.ID)
	assert.Equal(t, collection.Title, retrieved.Title)
	assert.Equal(t, collection.Description, retrieved.Description)
}

func TestCollectionService_GetCollection_NotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	collectionService := NewCollectionService(client, tableName, productService)

	_, err := collectionService.GetCollection(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Equal(t, ErrCollectionNotFound, err)
}

func TestCollectionService_UpdateCollection(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	collectionService := NewCollectionService(client, tableName, productService)

	// Create a collection
	collection := &models.Collection{
		Title:       "Test Collection",
		Description: "A test collection",
	}

	err := collectionService.CreateCollection(context.Background(), collection)
	require.NoError(t, err)

	// Update the collection
	collection.Title = "Updated Collection"
	collection.Description = "An updated collection"
	err = collectionService.UpdateCollection(context.Background(), collection)
	require.NoError(t, err)

	// Verify update
	updated, err := collectionService.GetCollection(context.Background(), collection.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Collection", updated.Title)
	assert.Equal(t, "An updated collection", updated.Description)
}

func TestCollectionService_DeleteCollection(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	collectionService := NewCollectionService(client, tableName, productService)

	// Create a collection
	collection := &models.Collection{
		Title:       "Test Collection",
		Description: "A test collection",
	}

	err := collectionService.CreateCollection(context.Background(), collection)
	require.NoError(t, err)

	// Delete the collection
	err = collectionService.DeleteCollection(context.Background(), collection.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = collectionService.GetCollection(context.Background(), collection.ID)
	assert.Error(t, err)
	assert.Equal(t, ErrCollectionNotFound, err)
}

func TestCollectionService_ListCollections(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	collectionService := NewCollectionService(client, tableName, productService)

	// Create multiple collections
	for i := 0; i < 5; i++ {
		collection := &models.Collection{
			Title:       "Test Collection " + string(rune('A'+i)),
			Description: "A test collection",
		}
		err := collectionService.CreateCollection(context.Background(), collection)
		require.NoError(t, err)
	}

	// List collections
	result, err := collectionService.ListCollections(context.Background(), 10, "", nil, "", "")
	require.NoError(t, err)
	assert.Len(t, result.Collections, 5)
}

func TestCollectionService_ListCollections_Pagination(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	collectionService := NewCollectionService(client, tableName, productService)

	// Create multiple collections
	for i := 0; i < 5; i++ {
		collection := &models.Collection{
			Title:       "Test Collection " + string(rune('A'+i)),
			Description: "A test collection",
		}
		err := collectionService.CreateCollection(context.Background(), collection)
		require.NoError(t, err)
	}

	// List with pagination
	result, err := collectionService.ListCollections(context.Background(), 2, "", nil, "", "")
	require.NoError(t, err)
	assert.Len(t, result.Collections, 2)
	assert.NotEmpty(t, result.NextCursor)

	// Get next page
	result2, err := collectionService.ListCollections(context.Background(), 2, result.NextCursor, nil, "", "")
	require.NoError(t, err)
	assert.Len(t, result2.Collections, 2)
}

func TestCollectionService_AddProductToCollection(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	collectionService := NewCollectionService(client, tableName, productService)

	// Create a collection
	collection := &models.Collection{
		Title:       "Test Collection",
		Description: "A test collection",
	}
	err := collectionService.CreateCollection(context.Background(), collection)
	require.NoError(t, err)

	// Create a product
	product := &models.Product{
		Title:       "Test Product",
		Description: "A test product",
		Status:      "active",
	}
	err = productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Add product to collection
	err = collectionService.AddProductToCollection(context.Background(), collection.ID, product.ID)
	require.NoError(t, err)

	// Verify product is in collection
	retrieved, err := collectionService.GetCollection(context.Background(), collection.ID)
	require.NoError(t, err)
	assert.Len(t, retrieved.Products, 1)
	assert.Equal(t, product.ID, retrieved.Products[0].ID)
}

func TestCollectionService_RemoveProductFromCollection(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	collectionService := NewCollectionService(client, tableName, productService)

	// Create a collection
	collection := &models.Collection{
		Title:       "Test Collection",
		Description: "A test collection",
	}
	err := collectionService.CreateCollection(context.Background(), collection)
	require.NoError(t, err)

	// Create a product
	product := &models.Product{
		Title:       "Test Product",
		Description: "A test product",
		Status:      "active",
	}
	err = productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Add product to collection
	err = collectionService.AddProductToCollection(context.Background(), collection.ID, product.ID)
	require.NoError(t, err)

	// Remove product from collection
	err = collectionService.RemoveProductFromCollection(context.Background(), collection.ID, product.ID)
	require.NoError(t, err)

	// Verify product is removed
	retrieved, err := collectionService.GetCollection(context.Background(), collection.ID)
	require.NoError(t, err)
	assert.Len(t, retrieved.Products, 0)
}

func TestCollectionService_CountCollections(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	collectionService := NewCollectionService(client, tableName, productService)

	// Create multiple collections
	for i := 0; i < 3; i++ {
		collection := &models.Collection{
			Title:       "Test Collection " + string(rune('A'+i)),
			Description: "A test collection",
		}
		err := collectionService.CreateCollection(context.Background(), collection)
		require.NoError(t, err)
	}

	// Count collections
	count, err := collectionService.CountCollections(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}
