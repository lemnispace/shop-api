package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body: func() string {
			body, err := json.Marshal(map[string]string{"message": "Hello from eCommerce API"})
			if err != nil {
				return ""
			}
			return string(body)
		}(),
	}, nil
}

func main() {
	lambda.Start(handler)
}
