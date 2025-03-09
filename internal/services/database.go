package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/lemnispace/shop-api/internal/models"
)

// Entity types - consistent naming for all entity types
const (
	EntityProduct              = "PRODUCT"
	EntityCart                 = "CART"
	EntityCollection           = "COLLECTION"
	EntityOrder                = "ORDER"
	EntityCustomer             = "CUSTOMER"
	EntityVariant              = "VARIANT"
	EntityImage                = "IMAGE"
	EntityCollectionProductRel = "COLLECTION_PRODUCT_RELATIONSHIP"
)

// ErrNotFound is returned when an item is not found in the database
var ErrNotFound = errors.New("item not found")

// Item represents a DynamoDB item with common attributes
type Item struct {
	PK         string `dynamodbav:"PK"`
	SK         string `dynamodbav:"SK"`
	GSI1PK     string `dynamodbav:"GSI1PK,omitempty"`
	GSI1SK     string `dynamodbav:"GSI1SK,omitempty"`
	GSI2PK     string `dynamodbav:"GSI2PK,omitempty"`
	GSI2SK     string `dynamodbav:"GSI2SK,omitempty"`
	EntityType string `dynamodbav:"EntityType"`
	Data       []byte `dynamodbav:"Data"`
}

// Key helper functions - all using consistent naming patterns

// ProductKey creates keys for a product
func ProductKey(productID string) (string, string) {
	return fmt.Sprintf("%s#%s", EntityProduct, productID), fmt.Sprintf("%s#%s", EntityProduct, productID)
}

// CartKey creates keys for a cart
func CartKey(cartID string) (string, string) {
	return fmt.Sprintf("%s#%s", EntityCart, cartID), fmt.Sprintf("%s#%s", EntityCart, cartID)
}

// CollectionKey creates keys for a collection
func CollectionKey(collectionID string) (string, string) {
	return fmt.Sprintf("%s#%s", EntityCollection, collectionID), fmt.Sprintf("%s#%s", EntityCollection, collectionID)
}

// OrderKey creates keys for an order
func OrderKey(orderID string) (string, string) {
	return fmt.Sprintf("%s#%s", EntityOrder, orderID), fmt.Sprintf("%s#%s", EntityOrder, orderID)
}

// CustomerKey creates keys for a customer
func CustomerKey(customerID string) (string, string) {
	return fmt.Sprintf("%s#%s", EntityCustomer, customerID), fmt.Sprintf("%s#%s", EntityCustomer, customerID)
}

// ProductVariantKey creates keys for a product variant
func ProductVariantKey(productID, variantID string) (string, string) {
	return fmt.Sprintf("%s#%s", EntityProduct, productID), fmt.Sprintf("%s#%s", EntityVariant, variantID)
}

// ProductImageKey creates keys for a product image
func ProductImageKey(productID, imageID string) (string, string) {
	return fmt.Sprintf("%s#%s", EntityProduct, productID), fmt.Sprintf("%s#%s", EntityImage, imageID)
}

// CollectionProductKey creates keys for a product in a collection
func CollectionProductKey(collectionID, productID string) (string, string) {
	return fmt.Sprintf("%s#%s", EntityCollection, collectionID), fmt.Sprintf("%s#%s", EntityProduct, productID)
}

type DynamoDB struct {
	client *dynamodb.Client
	table  string
}

func NewDynamoDB(client *dynamodb.Client, table string) *DynamoDB {
	return &DynamoDB{
		client: client,
		table:  table,
	}
}

// PutProduct stores a product in DynamoDB
func (d *DynamoDB) PutProduct(ctx context.Context, product *models.Product) error {
	pk, sk := ProductKey(product.ID)

	data, err := json.Marshal(product)
	if err != nil {
		return err
	}

	item := Item{
		PK:         pk,
		SK:         sk,
		GSI1PK:     fmt.Sprintf("SKU#%s", product.SKU),
		GSI1SK:     pk,
		EntityType: EntityProduct,
		Data:       data,
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return err
	}

	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.table),
		Item:      av,
	})

	return err
}

// GetProduct retrieves a product from DynamoDB
func (d *DynamoDB) GetProduct(ctx context.Context, id string) (*models.Product, error) {
	pk, sk := ProductKey(id)

	result, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.table),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, ErrNotFound
	}

	var item Item
	if err := attributevalue.UnmarshalMap(result.Item, &item); err != nil {
		return nil, err
	}

	var product models.Product
	if err := json.Unmarshal(item.Data, &product); err != nil {
		return nil, err
	}

	return &product, nil
}
