package main

import (
	"context"
	"net/http"

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
	lambda.Start(Handler)
}
