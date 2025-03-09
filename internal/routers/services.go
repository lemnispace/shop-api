package routers

import (
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/lemnispace/shop-api/internal/services"
)

// ProductService interface mirrors the internal services.ProductService interface
// This avoids circular imports while allowing the router to use the service
type ProductService interface {
	services.ProductService
}

// CollectionService interface mirrors the internal services.CollectionService interface
// This avoids circular imports while allowing the router to use the service
type CollectionService interface {
	services.CollectionService
}

// NewDynamoDBProductService creates a new product service backed by DynamoDB
func NewDynamoDBProductService(client *dynamodb.Client, tableName string) ProductService {
	return services.NewProductService(client, tableName)
}

// NewDynamoDBCollectionService creates a new collection service backed by DynamoDB
func NewDynamoDBCollectionService(client *dynamodb.Client, tableName string) CollectionService {
	productService := services.NewProductService(client, tableName)
	return services.NewCollectionService(client, tableName, productService)
}
