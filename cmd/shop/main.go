package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/lemnispace/shop-api/internal/routers"
	"github.com/lemnispace/shop-api/internal/services"
	"github.com/lemnispace/shop-api/internal/utils"
)

var router *http.ServeMux

func init() {
	router = routers.InitRouter()
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Convert the API Gateway request to an HTTP request
	httpRequest, err := utils.ProxyEventToHTTPRequest(req)
	if err != nil {
		return utils.NewErrorResponse(http.StatusBadRequest, err.Error()), nil
	}

	// Create a response recorder
	responseWriter := utils.NewResponseRecorder()

	// Serve the HTTP request
	router.ServeHTTP(responseWriter, httpRequest)

	// Convert the HTTP response to API Gateway response
	return responseWriter.GetProxyResponse()
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Configure services based on environment
	configureServices()

	// Initialize router
	router := routers.InitRouter()

	// Set up server with timeouts
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server gracefully stopped")
}

// configureServices determines storage configuration
func configureServices() {
	factory, err := createDynamoDBFactory()
	if err != nil {
		log.Fatalf("Failed to create DynamoDB client: %v", err)
	}

	// Set the DynamoDB service factory
	routers.SetServiceFactory(factory)

	// Log configuration
	if os.Getenv("AWS_ENDPOINT_URL") != "" {
		log.Println("Using local DynamoDB for storage")
	} else {
		log.Println("Using AWS DynamoDB for storage")
	}
}

// shouldUseDynamoDB checks environment flags to determine if DynamoDB should be used
func shouldUseDynamoDB() bool {
	// DynamoDB is always used now, this function is kept for backward compatibility
	return true
}

// createDynamoDBFactory creates a service factory for DynamoDB
func createDynamoDBFactory() (routers.ServiceFactory, error) {
	// Load AWS configuration
	cfg, err := loadAWSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create DynamoDB client
	client := dynamodb.NewFromConfig(cfg)

	// Get table name from environment or use default
	tableName := os.Getenv("DYNAMODB_TABLE")
	if tableName == "" {
		tableName = "ShopAPI"
	}

	// Verify table exists
	log.Printf("Verifying DynamoDB table exists: %s", tableName)
	tableExists, err := verifyTableExists(client, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to verify table existence: %w", err)
	}

	if !tableExists {
		log.Printf("DynamoDB table %s does not exist - will try to create it", tableName)
		if err := createTable(client, tableName); err != nil {
			return nil, fmt.Errorf("failed to create table: %w", err)
		}
	}

	return func() (services.ProductService, services.CollectionService) {
		productService := services.NewProductService(client, tableName)
		collectionService := services.NewCollectionService(client, tableName, productService)
		return productService, collectionService
	}, nil
}

// verifyTableExists checks if the DynamoDB table exists
func verifyTableExists(client *dynamodb.Client, tableName string) (bool, error) {
	ctx := context.Background()
	_, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})

	if err != nil {
		// Check if the error is because the table doesn't exist
		var notFoundErr *types.ResourceNotFoundException
		if errors.As(err, &notFoundErr) {
			return false, nil
		}
		// Some other error occurred
		return false, err
	}

	// Table exists
	return true, nil
}

// createTable creates a new DynamoDB table with the given name
func createTable(client *dynamodb.Client, tableName string) error {
	ctx := context.Background()
	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
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

	if err != nil {
		return err
	}

	log.Printf("Created DynamoDB table: %s", tableName)
	return nil
}

// loadAWSConfig loads the AWS configuration from environment or defaults
func loadAWSConfig() (aws.Config, error) {
	ctx := context.Background()

	// Check for local endpoint URL
	if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
		// For local development with DynamoDB Local
		customResolver := aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: endpoint}, nil
			},
		)
		return config.LoadDefaultConfig(ctx, config.WithEndpointResolverWithOptions(customResolver))
	}

	// Load normal AWS config
	return config.LoadDefaultConfig(ctx)
}
