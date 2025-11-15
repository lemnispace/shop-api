package services

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/lemnispace/shop-api/internal/models"
)

// FulfillmentService defines the interface for fulfillment operations
type FulfillmentService interface {
	// CreateFulfillment creates a new fulfillment record
	CreateFulfillment(ctx context.Context, input *models.FulfillmentInput) (*models.Fulfillment, error)

	// GetFulfillment retrieves a fulfillment by ID
	GetFulfillment(ctx context.Context, fulfillmentID string) (*models.Fulfillment, error)

	// UpdateFulfillmentStatus updates the status of a fulfillment
	UpdateFulfillmentStatus(ctx context.Context, fulfillmentID string, status models.FulfillmentStatus) error

	// GetOrderFulfillments retrieves all fulfillments for an order
	GetOrderFulfillments(ctx context.Context, orderID string) ([]*models.Fulfillment, error)

	// SubmitOrderToPrintful submits an order to Printful for fulfillment
	SubmitOrderToPrintful(ctx context.Context, order *models.Order) (*models.Fulfillment, error)
}

// FulfillmentServiceImpl implements the FulfillmentService interface
type FulfillmentServiceImpl struct {
	client          *dynamodb.Client
	tableName       string
	printfulClient  PrintfulService
	customerService CustomerService
}

// NewFulfillmentService creates a new fulfillment service
func NewFulfillmentService(client *dynamodb.Client, tableName string, printfulClient PrintfulService, customerService CustomerService) FulfillmentService {
	return &FulfillmentServiceImpl{
		client:          client,
		tableName:       tableName,
		printfulClient:  printfulClient,
		customerService: customerService,
	}
}

// CreateFulfillment creates a new fulfillment record
func (s *FulfillmentServiceImpl) CreateFulfillment(ctx context.Context, input *models.FulfillmentInput) (*models.Fulfillment, error) {
	fulfillmentID := "fulfillment_" + uuid.New().String()
	now := time.Now()

	fulfillment := &models.Fulfillment{
		ID:        fulfillmentID,
		OrderID:   input.OrderID,
		Status:    models.FulfillmentStatusPending,
		PartnerID: input.PartnerID,
		Items:     make([]models.FulfillmentItem, 0, len(input.Items)),
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Convert input items to fulfillment items
	for _, item := range input.Items {
		fulfillment.Items = append(fulfillment.Items, models.FulfillmentItem{
			ID:          "fitem_" + uuid.New().String(),
			OrderItemID: item.OrderItemID,
			Quantity:    item.Quantity,
		})
	}

	// Store in DynamoDB
	item := map[string]types.AttributeValue{
		"PK":             &types.AttributeValueMemberS{Value: fmt.Sprintf("FULFILLMENT#%s", fulfillmentID)},
		"SK":             &types.AttributeValueMemberS{Value: "METADATA"},
		"GSI1PK":         &types.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", input.OrderID)},
		"GSI1SK":         &types.AttributeValueMemberS{Value: fmt.Sprintf("FULFILLMENT#%s", fulfillmentID)},
		"EntityType":     &types.AttributeValueMemberS{Value: "FULFILLMENT"},
		"ID":             &types.AttributeValueMemberS{Value: fulfillmentID},
		"OrderID":        &types.AttributeValueMemberS{Value: input.OrderID},
		"Status":         &types.AttributeValueMemberS{Value: string(fulfillment.Status)},
		"PartnerID":      &types.AttributeValueMemberS{Value: input.PartnerID},
		"CreatedAt":      &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		"UpdatedAt":      &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
	}

	// Add items as a map
	if len(fulfillment.Items) > 0 {
		itemsBytes, err := attributevalue.Marshal(fulfillment.Items)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal items: %w", err)
		}
		item["Items"] = itemsBytes
	}

	_, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create fulfillment: %w", err)
	}

	return fulfillment, nil
}

// GetFulfillment retrieves a fulfillment by ID
func (s *FulfillmentServiceImpl) GetFulfillment(ctx context.Context, fulfillmentID string) (*models.Fulfillment, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("FULFILLMENT#%s", fulfillmentID)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get fulfillment: %w", err)
	}

	if result.Item == nil {
		return nil, ErrOrderNotFound // Reuse order not found error
	}

	var fulfillment models.Fulfillment
	if err := attributevalue.UnmarshalMap(result.Item, &fulfillment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal fulfillment: %w", err)
	}

	return &fulfillment, nil
}

// UpdateFulfillmentStatus updates the status of a fulfillment
func (s *FulfillmentServiceImpl) UpdateFulfillmentStatus(ctx context.Context, fulfillmentID string, status models.FulfillmentStatus) error {
	now := time.Now()

	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("FULFILLMENT#%s", fulfillmentID)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression: aws.String("SET #status = :status, UpdatedAt = :updatedAt"),
		ExpressionAttributeNames: map[string]string{
			"#status": "Status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":    &types.AttributeValueMemberS{Value: string(status)},
			":updatedAt": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to update fulfillment status: %w", err)
	}

	return nil
}

// GetOrderFulfillments retrieves all fulfillments for an order
func (s *FulfillmentServiceImpl) GetOrderFulfillments(ctx context.Context, orderID string) ([]*models.Fulfillment, error) {
	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk AND begins_with(GSI1SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", orderID)},
			":sk": &types.AttributeValueMemberS{Value: "FULFILLMENT#"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query order fulfillments: %w", err)
	}

	fulfillments := make([]*models.Fulfillment, 0, len(result.Items))
	for _, item := range result.Items {
		var fulfillment models.Fulfillment
		if err := attributevalue.UnmarshalMap(item, &fulfillment); err != nil {
			log.Printf("[WARN] Failed to unmarshal fulfillment: %v", err)
			continue
		}
		fulfillments = append(fulfillments, &fulfillment)
	}

	return fulfillments, nil
}

// SubmitOrderToPrintful submits an order to Printful for fulfillment
func (s *FulfillmentServiceImpl) SubmitOrderToPrintful(ctx context.Context, order *models.Order) (*models.Fulfillment, error) {
	if s.printfulClient == nil {
		return nil, fmt.Errorf("printful service not configured")
	}

	// Filter items that need Printful fulfillment
	printfulItems := make([]models.PrintfulOrderItem, 0)
	fulfillmentItemsInput := make([]models.FulfillmentItemInput, 0)

	for _, item := range order.Items {
		if item.FulfillmentData.PartnerID == "printful" {
			// Get product title from nested Product field
			itemName := ""
			if item.Product != nil {
				itemName = item.Product.Title
			}

			// Parse variant ID from string to int
			variantID, err := strconv.Atoi(item.FulfillmentData.PartnerVariantID)
			if err != nil {
				log.Printf("[WARN] Invalid Printful variant ID %s: %v", item.FulfillmentData.PartnerVariantID, err)
				continue
			}

			printfulItems = append(printfulItems, models.PrintfulOrderItem{
				VariantID:   variantID,
				Quantity:    item.Quantity,
				Name:        itemName,
				RetailPrice: fmt.Sprintf("%.2f", float64(item.Price)/100.0),
			})
			fulfillmentItemsInput = append(fulfillmentItemsInput, models.FulfillmentItemInput{
				OrderItemID: item.ID,
				Quantity:    item.Quantity,
			})
		}
	}

	if len(printfulItems) == 0 {
		return nil, fmt.Errorf("no items require Printful fulfillment")
	}

	// Fetch customer email
	var customerEmail string
	if s.customerService != nil && order.CustomerID != "" {
		customer, err := s.customerService.GetCustomer(ctx, order.CustomerID)
		if err != nil {
			log.Printf("[WARN] Failed to fetch customer email for order %s: %v", order.ID, err)
			// Continue without email - Printful will validate if it's required
		} else {
			customerEmail = customer.Email
		}
	}

	// Create Printful order request
	recipientName := fmt.Sprintf("%s %s", order.ShippingAddress.FirstName, order.ShippingAddress.LastName)
	printfulOrderReq := &models.PrintfulOrderRequest{
		ExternalID: order.ID,
		Recipient: models.PrintfulRecipient{
			Name:        recipientName,
			Address1:    order.ShippingAddress.Address1,
			Address2:    order.ShippingAddress.Address2,
			City:        order.ShippingAddress.City,
			StateCode:   order.ShippingAddress.Province,
			CountryCode: order.ShippingAddress.Country,
			Zip:         order.ShippingAddress.Zip,
			Phone:       order.ShippingAddress.Phone,
			Email:       customerEmail,
		},
		Items: printfulItems,
	}

	// Submit to Printful
	printfulOrder, err := s.printfulClient.CreateOrder(ctx, printfulOrderReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create Printful order: %w", err)
	}

	printfulOrderIDStr := fmt.Sprintf("%d", printfulOrder.ID)
	log.Printf("[INFO] Created Printful order: %s for order %s", printfulOrderIDStr, order.ID)

	// Create fulfillment record
	fulfillmentInput := &models.FulfillmentInput{
		OrderID:   order.ID,
		Items:     fulfillmentItemsInput,
		PartnerID: "printful",
	}

	fulfillment, err := s.CreateFulfillment(ctx, fulfillmentInput)
	if err != nil {
		log.Printf("[ERROR] Failed to create fulfillment record: %v", err)
		return nil, err
	}

	// Update with Printful order ID
	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("FULFILLMENT#%s", fulfillment.ID)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression: aws.String("SET PartnerOrderID = :partnerOrderID, #status = :status"),
		ExpressionAttributeNames: map[string]string{
			"#status": "Status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":partnerOrderID": &types.AttributeValueMemberS{Value: printfulOrderIDStr},
			":status":         &types.AttributeValueMemberS{Value: string(models.FulfillmentStatusProcessing)},
		},
	})
	if err != nil {
		log.Printf("[ERROR] Failed to update fulfillment with Printful order ID: %v", err)
	} else {
		fulfillment.PartnerOrderID = printfulOrderIDStr
		fulfillment.Status = models.FulfillmentStatusProcessing
	}

	return fulfillment, nil
}
