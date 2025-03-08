package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/lemnispace/shop-api/internal/routers"
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
	// Check if we're running locally or in AWS Lambda
	_, inLambda := os.LookupEnv("AWS_LAMBDA_RUNTIME_API")

	if inLambda {
		// Running in Lambda environment
		lambda.Start(Handler)
	} else {
		// Running locally - start HTTP server
		port := "8080"
		if p := os.Getenv("PORT"); p != "" {
			port = p
		}

		fmt.Printf("Starting local development server on port %s...\n", port)
		if err := http.ListenAndServe(":"+port, router); err != nil {
			log.Fatalf("Error starting server: %v", err)
		}
	}
}
