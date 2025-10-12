package routers

import (
	"context"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/lemnispace/shop-api/internal/utils"
)

// ProxyHandler converts API Gateway events to HTTP requests and processes them
// through our Gin router for AWS Lambda execution
func ProxyHandler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Ensure router is initialized
	router := InitRouter()

	// Convert the API Gateway request to an HTTP request
	httpRequest, err := utils.ProxyEventToHTTPRequest(req)
	if err != nil {
		return utils.NewErrorResponse(http.StatusBadRequest, err.Error()), nil
	}

	// Add context to the request
	httpRequest = httpRequest.WithContext(ctx)

	// Create a response recorder
	responseWriter := utils.NewResponseRecorder()

	// Serve the HTTP request - Gin implements http.Handler interface
	router.ServeHTTP(responseWriter, httpRequest)

	// Return the response
	return responseWriter.GetProxyResponse()
}
