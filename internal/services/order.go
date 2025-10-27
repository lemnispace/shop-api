package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/internal/utils"
)

var (
	ErrOrderNotFound      = fmt.Errorf("order not found")
	ErrCartEmpty          = fmt.Errorf("cart is empty")
	ErrInvalidOrderStatus = fmt.Errorf("invalid order status")
)

// OrderService defines the interface for order operations
type OrderService interface {
	CreateOrder(ctx context.Context, input *models.OrderInput) (*models.Order, error)
	GetOrder(ctx context.Context, orderID string) (*models.Order, error)
	ListOrders(ctx context.Context, limit int, cursor string, filters map[string]interface{}) (*OrderListResult, error)
	GetOrdersByCustomer(ctx context.Context, customerID string, limit int, cursor string) (*OrderListResult, error)
	UpdateOrderStatus(ctx context.Context, orderID string, status models.OrderStatus) error
	CancelOrder(ctx context.Context, orderID string) error
}

// OrderListResult represents a paginated list of orders
type OrderListResult struct {
	Orders     []*models.Order `json:"orders"`
	NextCursor string          `json:"nextCursor,omitempty"`
}

// DynamoDBOrderService implements OrderService using DynamoDB
type DynamoDBOrderService struct {
	db          *dynamodb.Client
	tableName   string
	cartService CartServiceInterface
}

// NewOrderService creates a new DynamoDB-backed order service
func NewOrderService(db *dynamodb.Client, tableName string, cartService CartServiceInterface) *DynamoDBOrderService {
	log.Printf("Initializing DynamoDB Order Service with table: %s", tableName)
	return &DynamoDBOrderService{
		db:          db,
		tableName:   tableName,
		cartService: cartService,
	}
}

// CreateOrder creates a new order from a cart
func (s *DynamoDBOrderService) CreateOrder(ctx context.Context, input *models.OrderInput) (*models.Order, error) {
	// 1. Validate and get cart
	cart, err := s.cartService.GetCart(ctx, input.CartID)
	if err != nil {
		if err == ErrCartNotFound {
			return nil, ErrCartNotFound
		}
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	// SECURITY: Verify cart ownership before creating order
	// This prevents users from creating orders from other customers' carts
	if cart.CustomerID != "" {
		// Cart has an owner - verify it matches the order customer
		if cart.CustomerID != input.CustomerID {
			log.Printf("[SECURITY] Attempted to create order for customer %s from cart belonging to customer %s",
				input.CustomerID, cart.CustomerID)
			return nil, fmt.Errorf("cannot create order: cart belongs to a different customer")
		}
	}
	// Note: If cart.CustomerID is empty (anonymous cart), we allow the order
	// and associate it with input.CustomerID

	// 2. Validate cart has items
	if len(cart.Items) == 0 {
		return nil, ErrCartEmpty
	}

	// 3. Create order from cart
	now := time.Now()
	orderID := fmt.Sprintf("ord_%d", now.UnixNano())

	order := &models.Order{
		ID:               orderID,
		CustomerID:       input.CustomerID,
		Items:            cart.Items,
		// TODO(finance): Switch monetary fields to fixed-precision integers (cents) or a decimal
		// type so we avoid float rounding issues when reconciling against Stripe amounts.
		Subtotal:         cart.Subtotal,
		Tax:              cart.EstimatedTax,
		Shipping:         cart.EstimatedShipping,
		TotalPrice:       cart.TotalPrice,
		Status:           models.OrderStatusPending,
		ShippingAddress:  input.ShippingAddress,
		BillingAddress:   input.BillingAddress,
		ShippingMethod:   input.ShippingMethod,
		PaymentMethod:    input.PaymentMethod,
		Fulfillments:     []models.Fulfillment{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// 4. Store order in DynamoDB
	orderItem := map[string]types.AttributeValue{
		"PK":                   &types.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", orderID)},
		"SK":                   &types.AttributeValueMemberS{Value: "METADATA"},
		"GSI1PK":               &types.AttributeValueMemberS{Value: fmt.Sprintf("CUSTOMER#%s", input.CustomerID)},
		"GSI1SK":               &types.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", now.Format(time.RFC3339))},
		"GSI2PK":               &types.AttributeValueMemberS{Value: fmt.Sprintf("STATUS#%s", order.Status)},
		"GSI2SK":               &types.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", orderID)},
		"ID":                   &types.AttributeValueMemberS{Value: orderID},
		"CustomerID":           &types.AttributeValueMemberS{Value: input.CustomerID},
		"Subtotal":             &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", order.Subtotal)},
		"Tax":                  &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", order.Tax)},
		"Shipping":             &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", order.Shipping)},
		"TotalPrice":           &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", order.TotalPrice)},
		"Status":               &types.AttributeValueMemberS{Value: string(order.Status)},
		"ShippingMethod":       &types.AttributeValueMemberS{Value: input.ShippingMethod},
		"PaymentMethod":        &types.AttributeValueMemberS{Value: input.PaymentMethod},
		"CreatedAt":            &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		"UpdatedAt":            &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		"EntityType":           &types.AttributeValueMemberS{Value: "ORDER"},
	}

	// Marshal addresses
	shippingAddr, err := attributevalue.Marshal(order.ShippingAddress)
	if err == nil {
		orderItem["ShippingAddress"] = shippingAddr
	}

	billingAddr, err := attributevalue.Marshal(order.BillingAddress)
	if err == nil {
		orderItem["BillingAddress"] = billingAddr
	}

	// Marshal items
	itemsAttr, err := attributevalue.Marshal(order.Items)
	if err == nil {
		orderItem["Items"] = itemsAttr
	}

	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      orderItem,
	})
	if err != nil {
		log.Printf("[ERROR] Failed to create order: %v", err)
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// 5. Store order items as separate records for easier querying
	for _, item := range order.Items {
		totalPrice := item.Price * float64(item.Quantity)
		itemRecord := map[string]types.AttributeValue{
			"PK":            &types.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", orderID)},
			"SK":            &types.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", item.ID)},
			"ProductID":     &types.AttributeValueMemberS{Value: item.ProductID},
			"VariantID":     &types.AttributeValueMemberS{Value: item.VariantID},
			"Quantity":      &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", item.Quantity)},
			"Price":         &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", item.Price)},
			"TotalPrice":    &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", totalPrice)},
			"EntityType":    &types.AttributeValueMemberS{Value: "ORDER_ITEM"},
		}

		_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(s.tableName),
			Item:      itemRecord,
		})
		if err != nil {
			log.Printf("[ERROR] Failed to store order item: %v", err)
			// Continue with other items even if one fails
		}
	}

	log.Printf("Successfully created order: %s for customer: %s", orderID, input.CustomerID)
	return order, nil
}

// GetOrder retrieves an order by ID
func (s *DynamoDBOrderService) GetOrder(ctx context.Context, orderID string) (*models.Order, error) {
	result, err := s.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", orderID)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		log.Printf("[ERROR] Failed to get order: %v", err)
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if result.Item == nil {
		log.Printf("[ERROR] Order not found with ID: %s", orderID)
		return nil, ErrOrderNotFound
	}

	var order models.Order
	err = attributevalue.UnmarshalMap(result.Item, &order)
	if err != nil {
		log.Printf("[ERROR] Failed to unmarshal order: %v", err)
		return nil, fmt.Errorf("failed to unmarshal order: %w", err)
	}

	return &order, nil
}

// ListOrders lists all orders with optional filters
func (s *DynamoDBOrderService) ListOrders(ctx context.Context, limit int, cursor string, filters map[string]interface{}) (*OrderListResult, error) {
	// Default limit
	if limit <= 0 {
		limit = 20
	}

	orders := make([]*models.Order, 0, limit)
	var lastEvaluatedKey map[string]types.AttributeValue

	// Handle cursor for initial pagination
	if cursor != "" {
		var err error
		lastEvaluatedKey, err = utils.DecodeCursor(cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
	}

	// Loop until we have enough orders or run out of items
	// This is necessary because DynamoDB applies Limit before FilterExpression
	for len(orders) < limit {
		input := &dynamodb.ScanInput{
			TableName:        aws.String(s.tableName),
			Limit:            aws.Int32(int32(limit * 2)), // Scan more items to account for filtering
			FilterExpression: aws.String("EntityType = :entityType AND SK = :metadata"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":entityType": &types.AttributeValueMemberS{Value: "ORDER"},
				":metadata":   &types.AttributeValueMemberS{Value: "METADATA"},
			},
		}

		if lastEvaluatedKey != nil {
			input.ExclusiveStartKey = lastEvaluatedKey
		}

		result, err := s.db.Scan(ctx, input)
		if err != nil {
			log.Printf("[ERROR] Failed to scan orders: %v", err)
			return nil, fmt.Errorf("failed to list orders: %w", err)
		}

		// Add results to orders list
		for _, item := range result.Items {
			if len(orders) >= limit {
				// We have enough orders, save the key for next cursor
				lastEvaluatedKey = map[string]types.AttributeValue{
					"PK": item["PK"],
					"SK": item["SK"],
				}
				break
			}

			var order models.Order
			if err := attributevalue.UnmarshalMap(item, &order); err != nil {
				log.Printf("[ERROR] Failed to unmarshal order: %v", err)
				continue
			}
			orders = append(orders, &order)
		}

		// Update lastEvaluatedKey for next iteration
		if result.LastEvaluatedKey != nil {
			lastEvaluatedKey = result.LastEvaluatedKey
		} else {
			// No more items to scan
			lastEvaluatedKey = nil
			break
		}

		// If we didn't get any items in this scan, we're done
		if len(result.Items) == 0 {
			break
		}
	}

	// Encode next cursor
	var nextCursor string
	if lastEvaluatedKey != nil {
		var err error
		nextCursor, err = utils.EncodeCursor(lastEvaluatedKey)
		if err != nil {
			log.Printf("[ERROR] Failed to encode cursor: %v", err)
		}
	}

	return &OrderListResult{
		Orders:     orders,
		NextCursor: nextCursor,
	}, nil
}

// GetOrdersByCustomer retrieves all orders for a specific customer
func (s *DynamoDBOrderService) GetOrdersByCustomer(ctx context.Context, customerID string, limit int, cursor string) (*OrderListResult, error) {
	// Default limit
	if limit <= 0 {
		limit = 20
	}

	input := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :customerPK AND begins_with(GSI1SK, :orderPrefix)"),
		FilterExpression:       aws.String("SK = :metadata"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":customerPK":  &types.AttributeValueMemberS{Value: fmt.Sprintf("CUSTOMER#%s", customerID)},
			":orderPrefix": &types.AttributeValueMemberS{Value: "ORDER#"},
			":metadata":    &types.AttributeValueMemberS{Value: "METADATA"},
		},
		Limit:            aws.Int32(int32(limit)),
		ScanIndexForward: aws.Bool(false), // Most recent first
	}

	// Handle cursor for pagination
	if cursor != "" {
		lastKey, err := utils.DecodeCursor(cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		input.ExclusiveStartKey = lastKey
	}

	result, err := s.db.Query(ctx, input)
	if err != nil {
		log.Printf("[ERROR] Failed to query orders for customer %s: %v", customerID, err)
		return nil, fmt.Errorf("failed to get customer orders: %w", err)
	}

	orders := make([]*models.Order, 0, len(result.Items))
	for _, item := range result.Items {
		var order models.Order
		if err := attributevalue.UnmarshalMap(item, &order); err != nil {
			log.Printf("[ERROR] Failed to unmarshal order: %v", err)
			continue
		}
		orders = append(orders, &order)
	}

	// Encode next cursor
	var nextCursor string
	if result.LastEvaluatedKey != nil {
		nextCursor, err = utils.EncodeCursor(result.LastEvaluatedKey)
		if err != nil {
			log.Printf("[ERROR] Failed to encode cursor: %v", err)
		}
	}

	return &OrderListResult{
		Orders:     orders,
		NextCursor: nextCursor,
	}, nil
}

// UpdateOrderStatus updates the status of an order
func (s *DynamoDBOrderService) UpdateOrderStatus(ctx context.Context, orderID string, status models.OrderStatus) error {
	// Validate status
	validStatuses := map[models.OrderStatus]bool{
		models.OrderStatusPending:   true,
		models.OrderStatusPaid:      true,
		models.OrderStatusFulfilled: true,
		models.OrderStatusShipped:   true,
		models.OrderStatusDelivered: true,
		models.OrderStatusCancelled: true,
	}

	if !validStatuses[status] {
		return ErrInvalidOrderStatus
	}

	now := time.Now()

	_, err := s.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", orderID)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression: aws.String("SET #status = :status, UpdatedAt = :updatedAt, GSI2PK = :gsi2pk"),
		ExpressionAttributeNames: map[string]string{
			"#status": "Status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":    &types.AttributeValueMemberS{Value: string(status)},
			":updatedAt": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
			":gsi2pk":    &types.AttributeValueMemberS{Value: fmt.Sprintf("STATUS#%s", status)},
		},
	})
	if err != nil {
		log.Printf("[ERROR] Failed to update order status: %v", err)
		return fmt.Errorf("failed to update order status: %w", err)
	}

	log.Printf("Updated order %s status to %s", orderID, status)
	return nil
}

// CancelOrder cancels an order (sets status to cancelled)
func (s *DynamoDBOrderService) CancelOrder(ctx context.Context, orderID string) error {
	// Get current order to check if cancellation is allowed
	order, err := s.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}

	// Don't allow cancellation of shipped or delivered orders
	if order.Status == models.OrderStatusShipped || order.Status == models.OrderStatusDelivered {
		return fmt.Errorf("cannot cancel order with status: %s", order.Status)
	}

	// Update status to cancelled
	return s.UpdateOrderStatus(ctx, orderID, models.OrderStatusCancelled)
}
