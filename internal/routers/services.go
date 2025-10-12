package routers

import (
	"context"
	"mime/multipart"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/internal/services"
)

// CustomizationService interface mirrors the internal services.CustomizationService interface
// This avoids circular imports while allowing the router to use the service
type CustomizationService interface {
	UploadImage(ctx context.Context, file multipart.File, fileHeader *multipart.FileHeader, userID, cartID, productID, variantID string) (*models.CustomizationImage, error)
	GetImage(ctx context.Context, imageID string) (*models.CustomizationImage, error)
	ProcessImage(ctx context.Context, imageID string, request models.ProcessImageRequest) (*models.ProcessImageResponse, error)
	DeleteImage(ctx context.Context, imageID string) error
	GetImagesByUserAndProduct(ctx context.Context, userID, productID, variantID string) ([]*models.CustomizationImage, error)
	LinkImageToCartItem(ctx context.Context, imageID, cartID, cartItemID string) error
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

// NewDynamoDBCartService creates a new cart service backed by DynamoDB
func NewDynamoDBCartService(client *dynamodb.Client, productService services.ProductService, tableName string) *services.CartService {
	return services.NewCartService(client, productService, tableName)
}

// NewS3Service creates a new S3 service with environment configuration
func NewS3Service() (S3Service, error) {
	return services.NewS3Service()
}

// NewCustomizationService creates a new customization service
func NewCustomizationService(client *dynamodb.Client, s3Service services.S3Service, tableName string) CustomizationService {
	return services.NewCustomizationService(client, s3Service, tableName)
}
