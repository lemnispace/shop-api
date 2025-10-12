package main

import (
	"context"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/lemnispace/shop-api/internal/handlers"
	"github.com/lemnispace/shop-api/internal/routers"
	"github.com/lemnispace/shop-api/internal/services"
)

var (
	initOnce sync.Once
	initErr  error
)

// initServices initializes all services once
func initServices() error {
	var err error
	initOnce.Do(func() {
		// Get table name from environment
		tableName := os.Getenv("DYNAMODB_TABLE_NAME")
		if tableName == "" {
			tableName = "ShopAPI" // Default for local development
			log.Printf("DYNAMODB_TABLE_NAME not set, using default: %s", tableName)
		}

		// Load AWS configuration
		cfg, configErr := config.LoadDefaultConfig(context.TODO())
		if configErr != nil {
			err = configErr
			log.Printf("ERROR: Failed to load AWS config: %v", configErr)
			return
		}

		// Create DynamoDB client with optional local endpoint
		var dbClient *dynamodb.Client
		dynamoEndpoint := os.Getenv("DYNAMODB_ENDPOINT")
		if dynamoEndpoint != "" {
			log.Printf("Using local DynamoDB endpoint: %s", dynamoEndpoint)
			dbClient = dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
				o.BaseEndpoint = &dynamoEndpoint
			})
		} else {
			dbClient = dynamodb.NewFromConfig(cfg)
		}
		log.Printf("DynamoDB client initialized for table: %s", tableName)

		// Initialize Product Service
		productService := services.NewProductService(dbClient, tableName)
		handlers.SetProductService(productService)
		log.Printf("Product service initialized and injected")

		// Initialize Collection Service
		collectionService := services.NewCollectionService(dbClient, tableName, productService)
		handlers.SetCollectionService(collectionService)
		log.Printf("Collection service initialized and injected")

		// Initialize Cart Service
		cartService := services.NewCartService(dbClient, productService, tableName)
		handlers.SetCartService(cartService)
		log.Printf("Cart service initialized and injected")

		// Initialize Order Service
		orderService := services.NewOrderService(dbClient, tableName, cartService)
		handlers.SetOrderService(orderService)
		log.Printf("Order service initialized and injected")

		// Initialize Payment Service
		stripeKey := os.Getenv("STRIPE_SECRET_KEY")
		if stripeKey == "" {
			log.Printf("WARNING: STRIPE_SECRET_KEY not set - payment endpoints will not work")
		} else {
			paymentService := services.NewPaymentService(stripeKey)
			handlers.SetPaymentService(paymentService)
			handlers.SetOrderServiceForPayments(orderService)
			log.Printf("Payment service initialized and injected")
		}

		// Initialize S3 Service for customizations
		s3Service, s3Err := services.NewS3Service()
		if s3Err != nil {
			log.Printf("WARNING: Failed to initialize S3 service: %v", s3Err)
			// Continue without S3 service - customization endpoints will fail
		} else {
			// Initialize Customization Service
			customizationService := services.NewCustomizationService(dbClient, s3Service, tableName)
			handlers.SetCustomizationService(customizationService)
			log.Printf("Customization service initialized and injected")
		}

		log.Printf("All services initialized successfully")
	})
	return err
}

// Handler is the Lambda handler for API Gateway events
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Initialize services on first request
	if err := initServices(); err != nil {
		log.Printf("ERROR: Failed to initialize services: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"error":"Internal server error - failed to initialize services"}`,
		}, nil
	}

	// Lambda entry point - converts API Gateway events to HTTP requests
	// and passes them through our router
	return routers.ProxyHandler(ctx, req)
}

// main is the entry point for the application when running locally
func main() {
	// Determine whether to run in Lambda or local mode based on environment
	if os.Getenv("RUN_LOCAL") == "true" {
		log.Printf("Starting shop-api in local development mode...")

		// Initialize services
		if err := initServices(); err != nil {
			log.Fatalf("Failed to initialize services: %v", err)
		}

		// Initialize router
		router := routers.InitRouter()

		// Get port from environment or use default
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}

		log.Printf("Server listening on port %s", port)
		log.Printf("API available at http://localhost:%s/v1", port)

		// Start HTTP server
		if err := router.Run(":" + port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	} else {
		// In Lambda environment, we should never reach this point
		// This is a safeguard to prevent accidental execution
		log.Fatalf("This binary is intended to run as a Lambda function. " +
			"Set RUN_LOCAL=true to run in local development mode.")
	}
}

