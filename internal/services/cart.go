package services

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/lemnispace/shop-api/internal/models"
)

var ErrCartNotFound = errors.New("cart not found")

type CartService struct {
	db             *dynamodb.Client
	tableName      string
	productService *ProductService
}

func NewCartService(db *dynamodb.Client, productService *ProductService) *CartService {
	return &CartService{
		db:             db,
		tableName:      "carts",
		productService: productService,
	}
}

func (s *CartService) calculateTotal(items []models.CartItem) float64 {
	var total float64
	for _, item := range items {
		total += item.Price * float64(item.Quantity)
	}
	return total
}

func (s *CartService) CreateCart(ctx context.Context) (*models.Cart, error) {
	cart := &models.Cart{
		ID:        uuid.New().String(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(), // Cart expires in 24 hours
	}

	item, err := attributevalue.MarshalMap(cart)
	if err != nil {
		return nil, err
	}

	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &s.tableName,
		Item:      item,
	})
	if err != nil {
		return nil, err
	}

	return cart, nil
}

func (s *CartService) AddItem(ctx context.Context, cartID string, input models.CartItemInput) error {
	// First verify product exists and is in stock
	product, err := s.productService.GetProduct(ctx, input.ProductID)
	if err != nil {
		return err
	}

	// Get current cart
	cart, err := s.GetCart(ctx, cartID)
	if err != nil {
		return err
	}

	// Add/update item
	newItem := models.CartItem{
		ProductID: input.ProductID,
		Quantity:  input.Quantity,
		Price:     product.Price, // You might want to get variant price if applicable
	}

	// Update cart
	cart.Items = append(cart.Items, newItem)
	cart.UpdatedAt = time.Now()
	cart.TotalPrice = s.calculateTotal(cart.Items)
	// Save cart
	item, err := attributevalue.MarshalMap(cart)
	if err != nil {
		return err
	}

	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &s.tableName,
		Item:      item,
	})

	return err
}

func (s *CartService) GetCart(ctx context.Context, id string) (*models.Cart, error) {
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
		return nil, ErrCartNotFound
	}

	var cart models.Cart
	err = attributevalue.UnmarshalMap(result.Item, &cart)
	if err != nil {
		return nil, err
	}

	return &cart, nil
}
