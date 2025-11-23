package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/lemnispace/shop-api/internal/models"
)

var (
	ErrCartNotFound          = errors.New("cart not found")
	ErrCartItemNotFound      = errors.New("cart item not found")
	ErrCartExpired           = errors.New("cart expired")
	ErrProductNotInStock     = errors.New("product not in stock")
	ErrInsufficientInventory = errors.New("insufficient inventory")
	ErrItemNotInCart         = errors.New("item not in cart")
	ErrInvalidQuantity       = errors.New("invalid quantity")
)

// CartService defines the interface for cart operations
type CartServiceInterface interface {
	CreateCart(ctx context.Context, customerID string) (*models.Cart, error)
	GetCart(ctx context.Context, id string) (*models.Cart, error)
	AddItem(ctx context.Context, cartID string, input models.CartItemInput) (*models.CartItem, error)
	UpdateItem(ctx context.Context, cartID string, itemID string, quantity int) (*models.CartItem, error)
	RemoveItem(ctx context.Context, cartID string, itemID string) error
	GetCheckoutURL(ctx context.Context, cartID string) (*models.CheckoutResponse, error)
	GetCartsByCustomer(ctx context.Context, customerID string, includeExpired bool) ([]*models.Cart, error)
}

// CartService implements the CartServiceInterface
type CartService struct {
	db              *dynamodb.Client
	tableName       string
	productService  ProductService
	taxRate         int64  // Tax rate in basis points (e.g., 900 = 9%)
	shippingRate    int64  // Base shipping rate in cents
	checkoutBaseURL string // Base URL for checkout service
}

// NewCartService creates a new cart service
func NewCartService(db *dynamodb.Client, productService ProductService, tableName string) *CartService {
	return &CartService{
		db:              db,
		tableName:       tableName,
		productService:  productService,
		taxRate:         900, // 9% tax rate (900 basis points)
		shippingRate:    599, // $5.99 base shipping rate (599 cents)
		checkoutBaseURL: "https://checkout.lemnispace.com/c/",
	}
}

// calculateSubtotal calculates the subtotal of the cart (before tax and shipping) in cents
func (s *CartService) calculateSubtotal(items []models.CartItem) int64 {
	var subtotal int64
	for _, item := range items {
		subtotal += item.Price * int64(item.Quantity)
	}
	return subtotal
}

// calculateTotalPrice calculates the total price of the cart including tax and shipping in cents
func (s *CartService) calculateTotalPrice(subtotal, tax, shipping int64) int64 {
	return subtotal + tax + shipping
}

// calculateTax calculates estimated tax for the cart in cents
func (s *CartService) calculateTax(subtotal int64) int64 {
	// taxRate is in basis points (e.g., 900 = 9%)
	// To calculate: (subtotal * taxRate) / 10000
	return (subtotal * s.taxRate) / 10000
}

// calculateShipping calculates estimated shipping for the cart items in cents
func (s *CartService) calculateShipping(items []models.CartItem) int64 {
	// Base shipping rate in cents
	shipping := s.shippingRate

	// Could be enhanced to calculate based on weight, dimensions, destination, etc.

	return shipping
}

// CreateCart creates a new cart, optionally associated with a customer
func (s *CartService) CreateCart(ctx context.Context, customerID string) (*models.Cart, error) {
	cart := &models.Cart{
		ID:                uuid.New().String(),
		CustomerID:        customerID,
		Items:             []models.CartItem{},
		Subtotal:          0,
		EstimatedTax:      0,
		EstimatedShipping: 0,
		TotalPrice:        0,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		ExpiresAt:         time.Now().Add(24 * time.Hour), // Cart expires in 24 hours
	}

	// Marshal cart data
	data, err := attributevalue.MarshalMap(cart)
	if err != nil {
		return nil, err
	}

	// Create keys using the single table design pattern
	pk, sk := CartKey(cart.ID)

	// Add key attributes to item
	data["PK"] = &types.AttributeValueMemberS{Value: pk}
	data["SK"] = &types.AttributeValueMemberS{Value: sk}
	data["EntityType"] = &types.AttributeValueMemberS{Value: EntityCart}

	// Add GSI keys for customer lookup if customer ID is provided
	if customerID != "" {
		data["GSI1PK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("CUSTOMER#%s", customerID)}
		data["GSI1SK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("CART#%s", cart.ID)}
	}

	// Save to DynamoDB
	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      data,
	})
	if err != nil {
		return nil, err
	}

	return cart, nil
}

// GetCart retrieves a cart by ID
func (s *CartService) GetCart(ctx context.Context, id string) (*models.Cart, error) {
	// Get keys using the single table design pattern
	pk, sk := CartKey(id)

	result, err := s.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
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

	// Check if cart has expired
	if time.Now().After(cart.ExpiresAt) {
		return nil, ErrCartExpired
	}

	// Enrich cart with product details
	for i, item := range cart.Items {
		product, err := s.productService.GetProduct(ctx, item.ProductID)
		if err == nil {
			// Find the variant
			var variantTitle string
			for _, variant := range product.Variants {
				if variant.ID == item.VariantID {
					variantTitle = variant.Title
					break
				}
			}

			// Add product details to cart item
			var imageURL string
			if len(product.Images) > 0 {
				imageURL = product.Images[0].URL
			}

			cart.Items[i].Product = &models.CartItemProduct{
				Title: product.Title,
				Image: imageURL,
			}

			if variantTitle != "" {
				cart.Items[i].Variant = &models.CartItemVariant{
					Title: variantTitle,
				}
			}
		}
	}

	return &cart, nil
}

// AddItem adds a new item to the cart
func (s *CartService) AddItem(ctx context.Context, cartID string, input models.CartItemInput) (*models.CartItem, error) {
	// Get the cart
	cart, err := s.GetCart(ctx, cartID)
	if err != nil {
		return nil, err
	}

	// Get the product to validate it exists and get price
	product, err := s.productService.GetProduct(ctx, input.ProductID)
	if err != nil {
		return nil, err
	}

	// Find the variant to get the correct price
	var variantPrice int64
	var variantFound bool

	if input.VariantID != "" {
		for _, variant := range product.Variants {
			if variant.ID == input.VariantID {
				variantPrice = variant.Price
				variantFound = true

				// Check if variant is in stock
				if variant.Inventory < input.Quantity {
					return nil, ErrProductNotInStock
				}
				break
			}
		}

		if !variantFound {
			return nil, ErrVariantNotFound
		}
	} else {
		// If no variant specified, use product price
		variantPrice = product.Price

		// Check if product is in stock
		if product.Inventory < input.Quantity {
			return nil, ErrProductNotInStock
		}
	}

	// Create the new cart item
	cartItem := models.CartItem{
		ID:        uuid.New().String(),
		ProductID: input.ProductID,
		VariantID: input.VariantID,
		Quantity:  input.Quantity,
		Price:     variantPrice,
	}

	// Add customization data if provided
	if input.CustomizationData != nil {
		cartItem.CustomizationData = input.CustomizationData
	}

	// Add product details
	var imageURL string
	if len(product.Images) > 0 {
		imageURL = product.Images[0].URL
	}

	cartItem.Product = &models.CartItemProduct{
		Title: product.Title,
		Image: imageURL,
	}

	// Add variant details if applicable
	if input.VariantID != "" {
		for _, variant := range product.Variants {
			if variant.ID == input.VariantID {
				cartItem.Variant = &models.CartItemVariant{
					Title: variant.Title,
				}
				// Add fulfillment data if available
				cartItem.FulfillmentData = variant.FulfillmentData
				break
			}
		}
	}

	// Add item to cart
	cart.Items = append(cart.Items, cartItem)

	// Update cart totals
	cart.Subtotal = s.calculateSubtotal(cart.Items)
	cart.EstimatedTax = s.calculateTax(cart.Subtotal)
	cart.EstimatedShipping = s.calculateShipping(cart.Items)
	cart.TotalPrice = s.calculateTotalPrice(cart.Subtotal, cart.EstimatedTax, cart.EstimatedShipping)
	cart.UpdatedAt = time.Now()

	// Save the updated cart
	err = s.saveCartToDynamoDB(ctx, cart)
	if err != nil {
		return nil, err
	}

	return &cartItem, nil
}

// UpdateItem updates the quantity of an item in the cart
func (s *CartService) UpdateItem(ctx context.Context, cartID string, itemID string, quantity int) (*models.CartItem, error) {
	// Get the cart
	cart, err := s.GetCart(ctx, cartID)
	if err != nil {
		return nil, err
	}

	// Find the item
	var itemIndex = -1
	var updatedItem models.CartItem

	for i, item := range cart.Items {
		if item.ID == itemID {
			itemIndex = i
			updatedItem = item
			break
		}
	}

	if itemIndex == -1 {
		return nil, ErrCartItemNotFound
	}

	// Verify that requested quantity is available
	product, err := s.productService.GetProduct(ctx, updatedItem.ProductID)
	if err != nil {
		return nil, err
	}

	if updatedItem.VariantID != "" {
		for _, variant := range product.Variants {
			if variant.ID == updatedItem.VariantID && variant.Inventory < quantity {
				return nil, ErrProductNotInStock
			}
		}
	} else if product.Inventory < quantity {
		return nil, ErrProductNotInStock
	}

	// Update the quantity
	cart.Items[itemIndex].Quantity = quantity
	updatedItem.Quantity = quantity

	// Recalculate cart totals
	cart.Subtotal = s.calculateSubtotal(cart.Items)
	cart.EstimatedTax = s.calculateTax(cart.Subtotal)
	cart.EstimatedShipping = s.calculateShipping(cart.Items)
	cart.TotalPrice = s.calculateTotalPrice(cart.Subtotal, cart.EstimatedTax, cart.EstimatedShipping)
	cart.UpdatedAt = time.Now()

	// Save the updated cart
	err = s.saveCartToDynamoDB(ctx, cart)
	if err != nil {
		return nil, err
	}

	return &updatedItem, nil
}

// RemoveItem removes an item from the cart
func (s *CartService) RemoveItem(ctx context.Context, cartID string, itemID string) error {
	// Get the cart
	cart, err := s.GetCart(ctx, cartID)
	if err != nil {
		return err
	}

	// Find and remove the item
	var found bool
	var updatedItems []models.CartItem

	for _, item := range cart.Items {
		if item.ID == itemID {
			found = true
		} else {
			updatedItems = append(updatedItems, item)
		}
	}

	if !found {
		return ErrCartItemNotFound
	}

	// Update the cart
	cart.Items = updatedItems

	// Recalculate cart totals
	cart.Subtotal = s.calculateSubtotal(cart.Items)
	cart.EstimatedTax = s.calculateTax(cart.Subtotal)
	cart.EstimatedShipping = s.calculateShipping(cart.Items)
	cart.TotalPrice = s.calculateTotalPrice(cart.Subtotal, cart.EstimatedTax, cart.EstimatedShipping)
	cart.UpdatedAt = time.Now()

	// Save the updated cart
	return s.saveCartToDynamoDB(ctx, cart)
}

// GetCheckoutURL generates a checkout URL for the cart
func (s *CartService) GetCheckoutURL(ctx context.Context, cartID string) (*models.CheckoutResponse, error) {
	// First, verify the cart exists
	_, err := s.GetCart(ctx, cartID)
	if err != nil {
		return nil, err
	}

	// Generate the checkout URL
	checkoutURL := fmt.Sprintf("%s%s", s.checkoutBaseURL, cartID)

	return &models.CheckoutResponse{
		CheckoutURL: checkoutURL,
	}, nil
}

// saveCartToDynamoDB internal helper function to save a cart to DynamoDB using the single table design
func (s *CartService) saveCartToDynamoDB(ctx context.Context, cart *models.Cart) error {
	// Marshal cart data
	data, err := attributevalue.MarshalMap(cart)
	if err != nil {
		return err
	}

	// Create keys using the single table design pattern
	pk, sk := CartKey(cart.ID)

	// Add key attributes to item
	data["PK"] = &types.AttributeValueMemberS{Value: pk}
	data["SK"] = &types.AttributeValueMemberS{Value: sk}
	data["EntityType"] = &types.AttributeValueMemberS{Value: EntityCart}

	// Add GSI keys for customer lookup if customer ID is provided
	if cart.CustomerID != "" {
		data["GSI1PK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("CUSTOMER#%s", cart.CustomerID)}
		data["GSI1SK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("CART#%s", cart.ID)}
	}

	// Save to DynamoDB
	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      data,
	})
	return err
}

// GetCartsByCustomer retrieves all carts for a specific customer, optionally filtering by active/expired
func (s *CartService) GetCartsByCustomer(ctx context.Context, customerID string, includeExpired bool) ([]*models.Cart, error) {
	now := time.Now()

	// Use GSI1 to query carts by customer
	input := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :customerPK"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":customerPK": &types.AttributeValueMemberS{Value: fmt.Sprintf("CUSTOMER#%s", customerID)},
		},
	}

	// If not including expired carts, add a filter expression
	if !includeExpired {
		input.FilterExpression = aws.String("ExpiresAt > :now")
		input.ExpressionAttributeValues[":now"] = &types.AttributeValueMemberS{
			Value: now.Format(time.RFC3339),
		}
	}

	result, err := s.db.Query(ctx, input)
	if err != nil {
		return nil, err
	}

	carts := make([]*models.Cart, 0)

	for _, item := range result.Items {
		var cart models.Cart
		if err := attributevalue.UnmarshalMap(item, &cart); err != nil {
			return nil, err
		}

		// Enrich each cart with product details (same as in GetCart)
		for i, cartItem := range cart.Items {
			product, err := s.productService.GetProduct(ctx, cartItem.ProductID)
			if err == nil {
				// Find the variant
				var variantTitle string
				for _, variant := range product.Variants {
					if variant.ID == cartItem.VariantID {
						variantTitle = variant.Title
						break
					}
				}

				// Add product details to cart item
				var imageURL string
				if len(product.Images) > 0 {
					imageURL = product.Images[0].URL
				}

				cart.Items[i].Product = &models.CartItemProduct{
					Title: product.Title,
					Image: imageURL,
				}

				if variantTitle != "" {
					cart.Items[i].Variant = &models.CartItemVariant{
						Title: variantTitle,
					}
				}
			}
		}

		carts = append(carts, &cart)
	}

	return carts, nil
}
