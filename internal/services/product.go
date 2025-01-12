package services

import (
	"context"

	"errors"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/lemnispace/shop-api/internal/models"
)

var ErrProductNotFound = errors.New("product not found")

type ProductService struct {
	db        *dynamodb.Client
	tableName string
}

func NewProductService(db *dynamodb.Client) *ProductService {
	return &ProductService{
		db:        db,
		tableName: "products",
	}
}

func (s *ProductService) GetProduct(ctx context.Context, id string) (*models.Product, error) {
	result, err := s.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, ErrProductNotFound
	}

	var product models.Product
	err = attributevalue.UnmarshalMap(result.Item, &product)
	if err != nil {
		return nil, err
	}

	return &product, nil
}

func (s *ProductService) ListProducts(ctx context.Context, limit int32) ([]models.Product, error) {
	result, err := s.db.Scan(ctx, &dynamodb.ScanInput{
		TableName: &s.tableName,
		Limit:     &limit,
	})
	if err != nil {
		return nil, err
	}

	var products []models.Product
	err = attributevalue.UnmarshalListOfMaps(result.Items, &products)
	if err != nil {
		return nil, err
	}

	return products, nil
}
