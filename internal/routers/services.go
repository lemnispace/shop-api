package routers

import (
	"context"
	"mime/multipart"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/internal/services"
)

// Service interface types to avoid circular imports

// ProductService interface mirrors the internal services.ProductService interface
type ProductService interface {
	GetProduct(ctx context.Context, id string) (*models.Product, error)
	CreateProduct(ctx context.Context, product *models.Product) error
	UpdateProduct(ctx context.Context, product *models.Product) error
	DeleteProduct(ctx context.Context, id string) error
	ListProducts(ctx context.Context, limit int, cursor string, filters map[string]interface{}, sortKey, sortOrder string) (*services.ProductListResult, error)
	CountProducts(ctx context.Context, filters map[string]interface{}) (int, error)
	ListProductVariants(ctx context.Context, productID string, limit int, cursor string) ([]models.ProductVariant, string, error)
	ListAllVariants(ctx context.Context, limit int, cursor string, filters map[string]interface{}, sortKey, sortOrder string) ([]models.ProductVariant, string, error)
	AddProductVariant(ctx context.Context, productID string, variant *models.ProductVariant) error
	UpdateProductVariant(ctx context.Context, productID string, variant *models.ProductVariant) error
	DeleteProductVariant(ctx context.Context, productID string, variantID string) error
	AddProductImage(ctx context.Context, productID string, image *models.Image) error
	AssociateImageWithVariant(ctx context.Context, productID string, variantID string, imageID string) error
}

// CollectionService interface mirrors the internal services.CollectionService interface
type CollectionService interface {
	GetCollection(ctx context.Context, id string) (*models.Collection, error)
	CreateCollection(ctx context.Context, collection *models.Collection) error
	UpdateCollection(ctx context.Context, collection *models.Collection) error
	DeleteCollection(ctx context.Context, id string) error
	ListCollections(ctx context.Context, limit int, cursor string, filters map[string]interface{}, sortKey, sortOrder string) (*services.CollectionListResult, error)
	CountCollections(ctx context.Context, filters map[string]interface{}) (int, error)
	AddProductToCollection(ctx context.Context, collectionID, productID string) error
	RemoveProductFromCollection(ctx context.Context, collectionID, productID string) error
	ListCollectionProducts(ctx context.Context, collectionID string, limit int, cursor string) ([]models.Product, string, error)
}

// S3Service interface for S3 operations
type S3Service interface {
	GenerateUploadURL(bucket, key string, expiresIn int) (string, error)
	GenerateDownloadURL(bucket, key string, expiresIn int) (string, error)
	DeleteObject(bucket, key string) error
}

// CustomizationService interface mirrors the internal services.CustomizationService interface
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
// Commented out due to compilation errors in s3.go
// func NewS3Service() (S3Service, error) {
// 	return services.NewS3Service()
// }

// NewCustomizationService creates a new customization service
// Commented out due to compilation errors in customization.go
// func NewCustomizationService(client *dynamodb.Client, s3Service services.S3Service, tableName string) CustomizationService {
// 	return services.NewCustomizationService(client, s3Service, tableName)
// }
