package services

import (
	"context"
	"testing"
	"time"

	"github.com/lemnispace/shop-api/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFulfillmentService_CreateAndGet(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	// Create a mock Printful service (nil is ok for this test)
	fulfillmentSvc := NewFulfillmentService(client, tableName, nil)

	// Create a fulfillment
	input := &models.FulfillmentInput{
		OrderID:   "order_test123",
		PartnerID: "printful",
		Items: []models.FulfillmentItemInput{
			{
				OrderItemID: "item_1",
				Quantity:    2,
			},
		},
	}

	fulfillment, err := fulfillmentSvc.CreateFulfillment(context.Background(), input)
	if err != nil {
		t.Fatalf("Failed to create fulfillment: %v", err)
	}

	assert.NotEmpty(t, fulfillment.ID)
	assert.Equal(t, input.OrderID, fulfillment.OrderID)
	assert.Equal(t, input.PartnerID, fulfillment.PartnerID)
	assert.Equal(t, models.FulfillmentStatusPending, fulfillment.Status)
	assert.Len(t, fulfillment.Items, 1)

	// Retrieve the fulfillment
	retrieved, err := fulfillmentSvc.GetFulfillment(context.Background(), fulfillment.ID)
	require.NoError(t, err)
	assert.Equal(t, fulfillment.ID, retrieved.ID)
}

func TestFulfillmentService_UpdateStatus(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	fulfillmentSvc := NewFulfillmentService(client, tableName, nil)

	// Create a fulfillment
	input := &models.FulfillmentInput{
		OrderID:   "order_test456",
		PartnerID: "printful",
		Items: []models.FulfillmentItemInput{
			{OrderItemID: "item_1", Quantity: 1},
		},
	}

	fulfillment, err := fulfillmentSvc.CreateFulfillment(context.Background(), input)
	if err != nil {
		t.Fatalf("Failed to create fulfillment: %v", err)
	}

	// Update status
	err = fulfillmentSvc.UpdateFulfillmentStatus(context.Background(), fulfillment.ID, models.FulfillmentStatusProcessing)
	if err != nil {
		t.Fatalf("Failed to update fulfillment status: %v", err)
	}

	// Verify status was updated
	retrieved, err := fulfillmentSvc.GetFulfillment(context.Background(), fulfillment.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FulfillmentStatusProcessing, retrieved.Status)
}

func TestFulfillmentService_GetOrderFulfillments(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	fulfillmentSvc := NewFulfillmentService(client, tableName, nil)

	orderID := "order_test789"

	// Create multiple fulfillments for the same order
	for i := 0; i < 3; i++ {
		input := &models.FulfillmentInput{
			OrderID:   orderID,
			PartnerID: "printful",
			Items: []models.FulfillmentItemInput{
				{OrderItemID: "item_1", Quantity: 1},
			},
		}

		_, err := fulfillmentSvc.CreateFulfillment(context.Background(), input)
		if err != nil {
			t.Fatalf("Failed to create fulfillment %d: %v", i, err)
		}
	}

	// Retrieve all fulfillments for the order
	fulfillments, err := fulfillmentSvc.GetOrderFulfillments(context.Background(), orderID)
	require.NoError(t, err)
	assert.Len(t, fulfillments, 3)

	// Verify all fulfillments belong to the correct order
	for _, f := range fulfillments {
		assert.Equal(t, orderID, f.OrderID)
	}
}

func TestFulfillmentService_SubmitOrderToPrintful_NoItems(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	// Create a mock Printful service
	mockPrintful := &MockPrintfulService{}
	fulfillmentSvc := NewFulfillmentService(client, tableName, mockPrintful)

	// Create an order with no Printful items
	order := &models.Order{
		ID:         "order_noitems",
		CustomerID: "cust_1",
		Items: []models.CartItem{
			{
				ID:        "item_1",
				ProductID: "prod_1",
				Quantity:  1,
				Price:     1000, // $10.00 in cents
				FulfillmentData: models.FulfillmentData{
					PartnerID: "other_partner", // Not printful
				},
			},
		},
		ShippingAddress: models.Address{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "123 Main St",
			City:      "Anytown",
			Province:  "CA",
			Country:   "US",
			Zip:       "12345",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Attempt to submit - should fail with no Printful items
	_, err := fulfillmentSvc.SubmitOrderToPrintful(context.Background(), order)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no items require Printful fulfillment")
}

// MockPrintfulService for testing
type MockPrintfulService struct {
	CreateOrderFunc func(ctx context.Context, order *models.PrintfulOrderRequest) (*models.PrintfulOrder, error)
}

func (m *MockPrintfulService) GetProducts(ctx context.Context) ([]models.PrintfulProduct, error) {
	return nil, nil
}

func (m *MockPrintfulService) GetProduct(ctx context.Context, productID int) (*models.PrintfulProduct, error) {
	return nil, nil
}

func (m *MockPrintfulService) GetVariant(ctx context.Context, variantID int) (*models.PrintfulVariant, error) {
	return nil, nil
}

func (m *MockPrintfulService) GetProductVariants(ctx context.Context, productID int) ([]models.PrintfulVariant, error) {
	return nil, nil
}

func (m *MockPrintfulService) CreateOrder(ctx context.Context, order *models.PrintfulOrderRequest) (*models.PrintfulOrder, error) {
	if m.CreateOrderFunc != nil {
		return m.CreateOrderFunc(ctx, order)
	}
	return &models.PrintfulOrder{
		ID:         12345,
		ExternalID: order.ExternalID,
		Status:     "draft",
	}, nil
}

func (m *MockPrintfulService) GetOrder(ctx context.Context, orderID string) (*models.PrintfulOrder, error) {
	return nil, nil
}

func (m *MockPrintfulService) ConfirmOrder(ctx context.Context, orderID string) (*models.PrintfulOrder, error) {
	return nil, nil
}

func (m *MockPrintfulService) CancelOrder(ctx context.Context, orderID string) (*models.PrintfulOrder, error) {
	return nil, nil
}

func (m *MockPrintfulService) SyncCatalog(ctx context.Context) (*models.PrintfulSyncJob, error) {
	return nil, nil
}

func (m *MockPrintfulService) ImportProduct(ctx context.Context, req *models.PrintfulProductImportRequest) (*models.Product, error) {
	return nil, nil
}
