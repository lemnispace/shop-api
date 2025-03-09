package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/lemnispace/shop-api/internal/locals"
	"github.com/lemnispace/shop-api/internal/routers"
	"github.com/lemnispace/shop-api/internal/services"
)

func init() {
	// Configure services on startup
	configureServices()
}

// Handler is the Lambda handler for API Gateway events
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Lambda entry point - converts API Gateway events to HTTP requests
	// and passes them through our router
	return routers.ProxyHandler(ctx, req)
}

// main is the entry point for the application when running locally
func main() {
	// Determine whether to run in Lambda or local mode based on environment
	if os.Getenv("RUN_LOCAL") == "true" {
		// Local development environment
		locals.RunLocalServer()
	} else {
		// In Lambda environment, we should never reach this point
		// This is a safeguard to prevent accidental execution
		log.Fatalf("This binary is intended to run as a Lambda function. " +
			"Set RUN_LOCAL=true to run in local development mode.")
	}
}

// configureServices initializes the services needed for the API
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

// createDynamoDBFactory creates a service factory for DynamoDB
func createDynamoDBFactory() (routers.ServiceFactory, error) {
	// Load AWS configuration
	cfg, err := loadAWSConfig()
	if err != nil {
		return nil, err
	}

	// Create DynamoDB client
	client := dynamodb.NewFromConfig(cfg)

	// Get table name from environment
	tableName := os.Getenv("DYNAMODB_TABLE")
	if tableName == "" {
		log.Fatalf("DYNAMODB_TABLE environment variable is required but not set")
	}

	// Return a factory function that creates services
	return func() (services.ProductService, services.CollectionService, *services.CartService) {
		// Create product service
		productService := services.NewProductService(client, tableName)

		// Create collection service using the product service
		collectionService := services.NewCollectionService(client, tableName, productService)

		// Create cart service using the product service
		cartService := services.NewCartService(client, productService, tableName)

		return productService, collectionService, cartService
	}, nil
}

// loadAWSConfig loads the AWS configuration from environment
func loadAWSConfig() (aws.Config, error) {
	// Default configuration options
	optFns := []func(*config.LoadOptions) error{
		config.WithRegion(os.Getenv("AWS_REGION")),
	}

	// If AWS_ENDPOINT_URL is set, use it for local development
	if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL: endpoint,
			}, nil
		})
		optFns = append(optFns, config.WithEndpointResolverWithOptions(customResolver))
	}

	// Load the config with our options
	return config.LoadDefaultConfig(context.Background(), optFns...)
}
