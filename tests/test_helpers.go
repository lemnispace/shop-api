package tests

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/lemnispace/shop-api/internal/services"
)

const (
	testTableName = "ShopAPITest"
)

// SetupDynamoDBForTesting creates a DynamoDB client and sets up a test table
func SetupDynamoDBForTesting() (*dynamodb.Client, error) {
	// Ensure we are always using the local DynamoDB for tests
	endpoint := "http://localhost:8000"

	// Use test credentials - this is crucial to avoid AWS credential errors
	testCreds := credentials.NewStaticCredentialsProvider("test", "test", "")

	// Configure AWS SDK with custom resolver for local endpoint and test credentials
	customResolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{URL: endpoint}, nil
		},
	)

	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(testCreds),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create DynamoDB client
	client := dynamodb.NewFromConfig(cfg)

	// Ensure the test table exists
	if err := ensureTestTableExists(client); err != nil {
		return nil, err
	}

	return client, nil
}

// ensureTestTableExists creates the test table if it doesn't exist
func ensureTestTableExists(client *dynamodb.Client) error {
	ctx := context.Background()

	// Check if table exists
	_, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(testTableName),
	})

	// If table exists, don't recreate it
	if err == nil {
		fmt.Printf("Table %s already exists, using existing table\n", testTableName)
		return nil
	}

	// Create the table
	_, err = client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(testTableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("PK"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("SK"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("PK"),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String("SK"),
				KeyType:       types.KeyTypeRange,
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})

	// If we get a ResourceInUseException, the table already exists
	var resourceInUse *types.ResourceInUseException
	if err != nil && !errors.As(err, &resourceInUse) {
		return fmt.Errorf("failed to create test table: %w", err)
	}

	// Wait for table to be active
	time.Sleep(2 * time.Second)

	return nil
}

// CreateTestProductService creates a product service for testing
func CreateTestProductService() (services.ProductService, error) {
	client, err := SetupDynamoDBForTesting()
	if err != nil {
		return nil, err
	}

	return services.NewProductService(client, testTableName), nil
}

// CreateTestCollectionService creates a collection service for testing
func CreateTestCollectionService(productService services.ProductService) (services.CollectionService, error) {
	client, err := SetupDynamoDBForTesting()
	if err != nil {
		return nil, err
	}

	// If productService is nil, create one
	if productService == nil {
		productService, err = CreateTestProductService()
		if err != nil {
			return nil, err
		}
	}

	return services.NewCollectionService(client, testTableName, productService), nil
}

// TeardownDynamoDBForTesting cleans up the test table
func TeardownDynamoDBForTesting(client *dynamodb.Client) error {
	if client == nil {
		return nil
	}

	_, err := client.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{
		TableName: aws.String(testTableName),
	})
	return err
}
