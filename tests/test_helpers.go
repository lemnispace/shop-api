package tests

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/lemnispace/shop-api/internal/services"
)

var (
	testTableName = "ShopAPITest"
	// Set to false to reduce test output verbosity
	verboseTestLogs = false

	// Mutex to coordinate table operations
	tableMutex sync.Mutex

	// Track if table has been set up
	tableIsSetup = false

	// Global client for tests
	globalClient *dynamodb.Client
)

// Helper function for controlled logging
func testLog(format string, args ...interface{}) {
	if verboseTestLogs {
		fmt.Printf(format+"\n", args...)
	}
}

// TestTableName returns the name of the test table
func TestTableName() string {
	return testTableName
}

// SetupDynamoDBForTesting creates a DynamoDB client and sets up a test table
func SetupDynamoDBForTesting() (*dynamodb.Client, error) {
	tableMutex.Lock()
	defer tableMutex.Unlock()

	// If we already have a global client, return it
	if globalClient != nil {
		return globalClient, nil
	}

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

	// Store the client globally
	globalClient = client

	// Ensure the test table exists
	if err := ensureTestTableExists(client); err != nil {
		return nil, err
	}

	// Mark table as set up
	tableIsSetup = true

	return client, nil
}

// ensureTestTableExists creates the test table if it doesn't exist
func ensureTestTableExists(client *dynamodb.Client) error {
	ctx := context.Background()

	// Check if table exists
	_, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(testTableName),
	})

	// If table exists and we're in a fresh test run, delete it for a clean state
	if err == nil && !tableIsSetup {
		testLog("Table %s exists, deleting for a clean test environment", testTableName)
		_, err = client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
			TableName: aws.String(testTableName),
		})
		if err != nil {
			// If we can't delete it, log the error but continue
			testLog("Warning: Could not delete existing table: %v", err)
		} else {
			// Wait for table to be deleted
			testLog("Waiting for table %s to be deleted...", testTableName)
			waiter := dynamodb.NewTableNotExistsWaiter(client)
			err = waiter.Wait(ctx, &dynamodb.DescribeTableInput{
				TableName: aws.String(testTableName),
			}, 30*time.Second)
			if err != nil {
				testLog("Warning: Timed out waiting for table deletion: %v", err)
			}
		}
	} else if err == nil {
		// Table exists and we've already set it up before, just use it
		testLog("Table %s exists and is already set up", testTableName)
		return nil
	}

	// Create the table with all required indexes for single table design
	testLog("Creating table %s with all required indexes", testTableName)
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
			{
				AttributeName: aws.String("GSI1PK"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("GSI1SK"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("GSI2PK"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("GSI2SK"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("EntityType"),
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
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("GSI1"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("GSI1PK"),
						KeyType:       types.KeyTypeHash,
					},
					{
						AttributeName: aws.String("GSI1SK"),
						KeyType:       types.KeyTypeRange,
					},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeAll,
				},
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(5),
					WriteCapacityUnits: aws.Int64(5),
				},
			},
			{
				IndexName: aws.String("GSI2"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("GSI2PK"),
						KeyType:       types.KeyTypeHash,
					},
					{
						AttributeName: aws.String("GSI2SK"),
						KeyType:       types.KeyTypeRange,
					},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeAll,
				},
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(5),
					WriteCapacityUnits: aws.Int64(5),
				},
			},
			{
				IndexName: aws.String("EntityTypeIndex"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("EntityType"),
						KeyType:       types.KeyTypeHash,
					},
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
		BillingMode: types.BillingModeProvisioned,
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
	})

	// Handle ResourceInUseException - table might be in the process of being created or deleted
	var resourceInUse *types.ResourceInUseException
	if err != nil && errors.As(err, &resourceInUse) {
		testLog("Table %s is already being created or deleted, waiting for it to be ready", testTableName)

		// Wait for the table to be ready
		maxAttempts := 20
		for attempt := 0; attempt < maxAttempts; attempt++ {
			time.Sleep(1 * time.Second)

			// Check if the table exists and is active
			desc, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
				TableName: aws.String(testTableName),
			})

			if err == nil && desc.Table != nil && desc.Table.TableStatus == types.TableStatusActive {
				testLog("Table %s is now active", testTableName)
				break
			}

			if attempt == maxAttempts-1 {
				return fmt.Errorf("timed out waiting for table to be ready")
			}
		}
	} else if err != nil {
		// If the table creation fails for other reasons, return the error
		return fmt.Errorf("failed to create test table: %w", err)
	} else {
		// Wait for table to be active using a simpler approach that doesn't print the table description
		testLog("Waiting for table %s to be active...", testTableName)

		// Use a simple polling approach instead of the waiter to avoid detailed output
		maxAttempts := 20
		for attempt := 0; attempt < maxAttempts; attempt++ {
			// Check if the table is active
			desc, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
				TableName: aws.String(testTableName),
			})

			if err == nil && desc.Table != nil && desc.Table.TableStatus == types.TableStatusActive {
				testLog("Table %s is now active", testTableName)
				break
			}

			if attempt == maxAttempts-1 {
				if err != nil {
					return fmt.Errorf("table creation verification failed: %w", err)
				}
				return fmt.Errorf("timed out waiting for table to become active")
			}

			time.Sleep(500 * time.Millisecond)
		}
	}

	// Verify the table is fully operational by performing a simple operation
	testLog("Verifying table %s is fully operational...", testTableName)
	testItem := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: "TEST#VERIFY"},
		"SK": &types.AttributeValueMemberS{Value: "TEST#VERIFY"},
	}

	// Try to put and then delete a test item to verify the table is operational
	_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(testTableName),
		Item:      testItem,
	})
	if err != nil {
		return fmt.Errorf("table verification failed - could not put test item: %w", err)
	}

	// Delete the test item
	_, err = client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(testTableName),
		Key: map[string]types.AttributeValue{
			"PK": testItem["PK"],
			"SK": testItem["SK"],
		},
	})
	if err != nil {
		testLog("Warning: Could not delete test item: %v", err)
	}

	testLog("Table %s is now active and ready for testing", testTableName)
	return nil
}

// ClearTestTable removes all data from the test table but keeps the table structure
func ClearTestTable(client *dynamodb.Client) error {
	if client == nil {
		return nil
	}

	tableMutex.Lock()
	defer tableMutex.Unlock()

	ctx := context.Background()

	// Scan to get all items
	resp, err := client.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(testTableName),
	})

	if err != nil {
		return fmt.Errorf("failed to scan table for clearing: %w", err)
	}

	// Delete all items individually
	for _, item := range resp.Items {
		_, err := client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
			TableName: aws.String(testTableName),
			Key: map[string]types.AttributeValue{
				"PK": item["PK"],
				"SK": item["SK"],
			},
		})
		if err != nil {
			testLog("Warning: Could not delete item during table clear: %v", err)
		}
	}

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
	// Instead of deleting the table after every test, we'll just clear the data
	// This helps avoid the "table does not exist" errors between tests
	return ClearTestTable(client)
}

// FinalCleanup performs the final cleanup after all tests are done
func FinalCleanup() {
	if globalClient == nil {
		return
	}

	tableMutex.Lock()
	defer tableMutex.Unlock()

	ctx := context.Background()

	// Check if the table exists before trying to delete it
	_, err := globalClient.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(testTableName),
	})

	if err != nil {
		// Table does not exist, nothing to clean up
		testLog("Table %s does not exist, nothing to clean up", testTableName)
		return
	}

	// Delete the table
	testLog("Deleting table %s", testTableName)
	_, err = globalClient.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(testTableName),
	})

	if err != nil {
		testLog("Failed to delete test table: %v", err)
		return
	}

	// Wait for table to be deleted
	testLog("Waiting for table %s to be deleted...", testTableName)
	waiter := dynamodb.NewTableNotExistsWaiter(globalClient)
	err = waiter.Wait(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(testTableName),
	}, 30*time.Second)

	if err != nil {
		testLog("Warning: Timed out waiting for table %s to be deleted: %v", testTableName, err)
	} else {
		testLog("Table %s has been successfully deleted", testTableName)
	}

	// Reset global state
	globalClient = nil
	tableIsSetup = false
}
