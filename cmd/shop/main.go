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
		// TODO: Implement local development environment
	} else {
		// In Lambda environment, we should never reach this point
		// This is a safeguard to prevent accidental execution
		log.Fatalf("This binary is intended to run as a Lambda function. " +
			"Set RUN_LOCAL=true to run in local development mode.")
	}
}

