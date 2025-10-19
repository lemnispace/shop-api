package services

import (
	"context"
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/lemnispace/shop-api/internal/models"
)

var (
	ErrCollectionNotFound = errors.New("collection not found")
)

// CollectionListResult represents the result of a collection list operation with pagination
type CollectionListResult struct {
	Collections []models.Collection
	NextCursor  string
}

// CollectionService defines the interface for collection operations
type CollectionService interface {
	GetCollection(ctx context.Context, id string) (*models.Collection, error)
	CreateCollection(ctx context.Context, collection *models.Collection) error
	UpdateCollection(ctx context.Context, collection *models.Collection) error
	DeleteCollection(ctx context.Context, id string) error
	ListCollections(ctx context.Context, limit int, cursor string, filters map[string]interface{}, sortKey, sortOrder string) (*CollectionListResult, error)
	CountCollections(ctx context.Context, filters map[string]interface{}) (int, error)
	CountCollectionProducts(ctx context.Context, collectionID string) (int, error)
	AddProductToCollection(ctx context.Context, collectionID, productID string) error
	RemoveProductFromCollection(ctx context.Context, collectionID, productID string) error
	ListCollectionProducts(ctx context.Context, collectionID string, limit int, cursor string) ([]models.Product, string, error)
}

// DynamoDBCollectionService is an implementation of CollectionService using DynamoDB
type DynamoDBCollectionService struct {
	db             *dynamodb.Client
	tableName      string
	productService ProductService
}

// NewCollectionService creates a new DynamoDB collection service
func NewCollectionService(db *dynamodb.Client, tableName string, productService ProductService) *DynamoDBCollectionService {
	if db == nil {
		log.Printf("WARNING: DynamoDB client is nil in NewCollectionService")
	}
	if tableName == "" {
		log.Printf("WARNING: Empty table name in NewCollectionService")
		tableName = "ShopAPI" // Default table name
	}

	log.Printf("Initializing DynamoDB Collection Service with table: %s", tableName)

	return &DynamoDBCollectionService{
		db:             db,
		tableName:      tableName,
		productService: productService,
	}
}
