package services

import (
	"context"
	"testing"
	"time"

	"github.com/lemnispace/shop-api/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderService_CreateOrder(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)
	orderService := NewOrderService(client, tableName, cartService)

	// Create a product and variant
	product := &models.Product{
		Title:       "Test Product",
		Description: "Test Description",
		Status:      "active",
		Variants: []models.ProductVariant{
			{
				Title:     "Test Variant",
				Price:     2999, // $29.99 in cents
				SKU:       "TEST-SKU",
				Inventory: 100, // Set inventory
			},
		},
	}
	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Create a cart with items
	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	// Add item to cart
	cartItem, err := cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
		ProductID: product.ID,
		VariantID: product.Variants[0].ID,
		Quantity:  2,
	})
	require.NoError(t, err)
	assert.NotNil(t, cartItem)

	// Create order from cart
	orderInput := &models.OrderInput{
		CartID:     cart.ID,
		CustomerID: "cust_123",
		ShippingAddress: models.Address{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "123 Main St",
			City:      "Anytown",
			Province:  "CA",
			Country:   "US",
			Zip:       "12345",
		},
		BillingAddress: models.Address{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "123 Main St",
			City:      "Anytown",
			Province:  "CA",
			Country:   "US",
			Zip:       "12345",
		},
		ShippingMethod: "standard",
		PaymentMethod:  "credit_card",
	}

	order, err := orderService.CreateOrder(context.Background(), orderInput)
	require.NoError(t, err)
	assert.NotEmpty(t, order.ID)
	assert.Equal(t, "cust_123", order.CustomerID)
	assert.Equal(t, models.OrderStatusPending, order.Status)
	assert.Len(t, order.Items, 1)
	assert.Equal(t, 2, order.Items[0].Quantity)
	assert.NotZero(t, order.CreatedAt)
	assert.NotZero(t, order.UpdatedAt)
}

func TestOrderService_CreateOrder_EmptyCart(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)
	orderService := NewOrderService(client, tableName, cartService)

	// Create empty cart
	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	// Try to create order from empty cart
	orderInput := &models.OrderInput{
		CartID:     cart.ID,
		CustomerID: "cust_123",
		ShippingAddress: models.Address{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "123 Main St",
			City:      "Anytown",
			Province:  "CA",
			Country:   "US",
			Zip:       "12345",
		},
		BillingAddress: models.Address{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "123 Main St",
			City:      "Anytown",
			Province:  "CA",
			Country:   "US",
			Zip:       "12345",
		},
	}

	_, err = orderService.CreateOrder(context.Background(), orderInput)
	assert.Error(t, err)
	assert.Equal(t, ErrCartEmpty, err)
}

func TestOrderService_CreateOrder_CartNotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)
	orderService := NewOrderService(client, tableName, cartService)

	// Try to create order with non-existent cart
	orderInput := &models.OrderInput{
		CartID:     "nonexistent",
		CustomerID: "cust_123",
		ShippingAddress: models.Address{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "123 Main St",
			City:      "Anytown",
			Province:  "CA",
			Country:   "US",
			Zip:       "12345",
		},
		BillingAddress: models.Address{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "123 Main St",
			City:      "Anytown",
			Province:  "CA",
			Country:   "US",
			Zip:       "12345",
		},
	}

	_, err := orderService.CreateOrder(context.Background(), orderInput)
	assert.Error(t, err)
	assert.Equal(t, ErrCartNotFound, err)
}

func TestOrderService_GetOrder(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)
	orderService := NewOrderService(client, tableName, cartService)

	// Create product, cart, and order
	product := &models.Product{
		Title:       "Test Product",
		Description: "Test Description",
		Status:      "active",
		Variants: []models.ProductVariant{
			{
				Title:     "Test Variant",
				Price:     2999, // $29.99 in cents
				SKU:       "TEST-SKU",
				Inventory: 100,
			},
		},
	}
	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	_, err = cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
		ProductID: product.ID,
		VariantID: product.Variants[0].ID,
		Quantity:  1,
	})
	require.NoError(t, err)

	orderInput := &models.OrderInput{
		CartID:     cart.ID,
		CustomerID: "cust_123",
		ShippingAddress: models.Address{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "123 Main St",
			City:      "Anytown",
			Province:  "CA",
			Country:   "US",
			Zip:       "12345",
		},
		BillingAddress: models.Address{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "123 Main St",
			City:      "Anytown",
			Province:  "CA",
			Country:   "US",
			Zip:       "12345",
		},
	}

	createdOrder, err := orderService.CreateOrder(context.Background(), orderInput)
	require.NoError(t, err)

	// Get the order
	retrievedOrder, err := orderService.GetOrder(context.Background(), createdOrder.ID)
	require.NoError(t, err)
	assert.Equal(t, createdOrder.ID, retrievedOrder.ID)
	assert.Equal(t, createdOrder.CustomerID, retrievedOrder.CustomerID)
	assert.Equal(t, createdOrder.Status, retrievedOrder.Status)
	assert.Len(t, retrievedOrder.Items, 1)
}

func TestOrderService_GetOrder_NotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)
	orderService := NewOrderService(client, tableName, cartService)

	_, err := orderService.GetOrder(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Equal(t, ErrOrderNotFound, err)
}

func TestOrderService_UpdateOrderStatus(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)
	orderService := NewOrderService(client, tableName, cartService)

	// Create order
	product := &models.Product{
		Title:       "Test Product",
		Description: "Test Description",
		Status:      "active",
		Variants: []models.ProductVariant{
			{
				Title:     "Test Variant",
				Price:     2999, // $29.99 in cents
				SKU:       "TEST-SKU",
				Inventory: 100,
			},
		},
	}
	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	_, err = cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
		ProductID: product.ID,
		VariantID: product.Variants[0].ID,
		Quantity:  1,
	})
	require.NoError(t, err)

	orderInput := &models.OrderInput{
		CartID:     cart.ID,
		CustomerID: "cust_123",
		ShippingAddress: models.Address{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "123 Main St",
			City:      "Anytown",
			Province:  "CA",
			Country:   "US",
			Zip:       "12345",
		},
		BillingAddress: models.Address{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "123 Main St",
			City:      "Anytown",
			Province:  "CA",
			Country:   "US",
			Zip:       "12345",
		},
	}

	order, err := orderService.CreateOrder(context.Background(), orderInput)
	require.NoError(t, err)
	assert.Equal(t, models.OrderStatusPending, order.Status)

	// Update status to paid
	err = orderService.UpdateOrderStatus(context.Background(), order.ID, models.OrderStatusPaid)
	require.NoError(t, err)

	// Verify status updated
	updatedOrder, err := orderService.GetOrder(context.Background(), order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.OrderStatusPaid, updatedOrder.Status)
}

func TestOrderService_UpdateOrderStatus_Invalid(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)
	orderService := NewOrderService(client, tableName, cartService)

	err := orderService.UpdateOrderStatus(context.Background(), "ord_123", models.OrderStatus("invalid"))
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidOrderStatus, err)
}

func TestOrderService_CancelOrder(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)
	orderService := NewOrderService(client, tableName, cartService)

	// Create order
	product := &models.Product{
		Title:       "Test Product",
		Description: "Test Description",
		Status:      "active",
		Variants: []models.ProductVariant{
			{
				Title:     "Test Variant",
				Price:     2999, // $29.99 in cents
				SKU:       "TEST-SKU",
				Inventory: 100,
			},
		},
	}
	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	_, err = cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
		ProductID: product.ID,
		VariantID: product.Variants[0].ID,
		Quantity:  1,
	})
	require.NoError(t, err)

	orderInput := &models.OrderInput{
		CartID:     cart.ID,
		CustomerID: "cust_123",
		ShippingAddress: models.Address{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "123 Main St",
			City:      "Anytown",
			Province:  "CA",
			Country:   "US",
			Zip:       "12345",
		},
		BillingAddress: models.Address{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "123 Main St",
			City:      "Anytown",
			Province:  "CA",
			Country:   "US",
			Zip:       "12345",
		},
	}

	order, err := orderService.CreateOrder(context.Background(), orderInput)
	require.NoError(t, err)

	// Cancel order
	err = orderService.CancelOrder(context.Background(), order.ID)
	require.NoError(t, err)

	// Verify status is cancelled
	cancelledOrder, err := orderService.GetOrder(context.Background(), order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.OrderStatusCancelled, cancelledOrder.Status)
}

func TestOrderService_CancelOrder_AlreadyShipped(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)
	orderService := NewOrderService(client, tableName, cartService)

	// Create order
	product := &models.Product{
		Title:       "Test Product",
		Description: "Test Description",
		Status:      "active",
		Variants: []models.ProductVariant{
			{
				Title:     "Test Variant",
				Price:     2999, // $29.99 in cents
				SKU:       "TEST-SKU",
				Inventory: 100,
			},
		},
	}
	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	_, err = cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
		ProductID: product.ID,
		VariantID: product.Variants[0].ID,
		Quantity:  1,
	})
	require.NoError(t, err)

	orderInput := &models.OrderInput{
		CartID:     cart.ID,
		CustomerID: "cust_123",
		ShippingAddress: models.Address{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "123 Main St",
			City:      "Anytown",
			Province:  "CA",
			Country:   "US",
			Zip:       "12345",
		},
		BillingAddress: models.Address{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "123 Main St",
			City:      "Anytown",
			Province:  "CA",
			Country:   "US",
			Zip:       "12345",
		},
	}

	order, err := orderService.CreateOrder(context.Background(), orderInput)
	require.NoError(t, err)

	// Update to shipped status
	err = orderService.UpdateOrderStatus(context.Background(), order.ID, models.OrderStatusShipped)
	require.NoError(t, err)

	// Try to cancel shipped order
	err = orderService.CancelOrder(context.Background(), order.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot cancel order")
}

func TestOrderService_GetOrdersByCustomer(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)
	orderService := NewOrderService(client, tableName, cartService)

	// Create product
	product := &models.Product{
		Title:       "Test Product",
		Description: "Test Description",
		Status:      "active",
		Variants: []models.ProductVariant{
			{
				Title:     "Test Variant",
				Price:     2999, // $29.99 in cents
				SKU:       "TEST-SKU",
				Inventory: 100,
			},
		},
	}
	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Create 3 orders for customer 1
	for i := 0; i < 3; i++ {
		cart, err := cartService.CreateCart(context.Background(), "cust_1")
		require.NoError(t, err)

		_, err = cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
			ProductID: product.ID,
			VariantID: product.Variants[0].ID,
			Quantity:  1,
		})
		require.NoError(t, err)

		orderInput := &models.OrderInput{
			CartID:     cart.ID,
			CustomerID: "cust_1",
			ShippingAddress: models.Address{
				FirstName: "John",
				LastName:  "Doe",
				Address1:  "123 Main St",
				City:      "Anytown",
				Province:  "CA",
				Country:   "US",
				Zip:       "12345",
			},
			BillingAddress: models.Address{
				FirstName: "John",
				LastName:  "Doe",
				Address1:  "123 Main St",
				City:      "Anytown",
				Province:  "CA",
				Country:   "US",
				Zip:       "12345",
			},
		}

		_, err = orderService.CreateOrder(context.Background(), orderInput)
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps
	}

	// Create 2 orders for customer 2
	for i := 0; i < 2; i++ {
		cart, err := cartService.CreateCart(context.Background(), "cust_2")
		require.NoError(t, err)

		_, err = cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
			ProductID: product.ID,
			VariantID: product.Variants[0].ID,
			Quantity:  1,
		})
		require.NoError(t, err)

		orderInput := &models.OrderInput{
			CartID:     cart.ID,
			CustomerID: "cust_2",
			ShippingAddress: models.Address{
				FirstName: "Jane",
				LastName:  "Smith",
				Address1:  "456 Oak St",
				City:      "Somewhere",
				Province:  "NY",
				Country:   "US",
				Zip:       "67890",
			},
			BillingAddress: models.Address{
				FirstName: "Jane",
				LastName:  "Smith",
				Address1:  "456 Oak St",
				City:      "Somewhere",
				Province:  "NY",
				Country:   "US",
				Zip:       "67890",
			},
		}

		_, err = orderService.CreateOrder(context.Background(), orderInput)
		require.NoError(t, err)
	}

	// Get orders for customer 1
	result, err := orderService.GetOrdersByCustomer(context.Background(), "cust_1", 10, "")
	require.NoError(t, err)
	assert.Len(t, result.Orders, 3)

	// Verify all orders belong to customer 1
	for _, order := range result.Orders {
		assert.Equal(t, "cust_1", order.CustomerID)
	}

	// Get orders for customer 2
	result, err = orderService.GetOrdersByCustomer(context.Background(), "cust_2", 10, "")
	require.NoError(t, err)
	assert.Len(t, result.Orders, 2)

	// Verify all orders belong to customer 2
	for _, order := range result.Orders {
		assert.Equal(t, "cust_2", order.CustomerID)
	}
}

func TestOrderService_ListOrders_Pagination(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)
	orderService := NewOrderService(client, tableName, cartService)

	// Create product
	product := &models.Product{
		Title:       "Test Product",
		Description: "Test Description",
		Status:      "active",
		Variants: []models.ProductVariant{
			{
				Title:     "Test Variant",
				Price:     2999, // $29.99 in cents
				SKU:       "TEST-SKU",
				Inventory: 100,
			},
		},
	}
	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Create 5 orders
	for i := 0; i < 5; i++ {
		cart, err := cartService.CreateCart(context.Background(), "cust_123")
		require.NoError(t, err)

		_, err = cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
			ProductID: product.ID,
			VariantID: product.Variants[0].ID,
			Quantity:  1,
		})
		require.NoError(t, err)

		orderInput := &models.OrderInput{
			CartID:     cart.ID,
			CustomerID: "cust_123",
			ShippingAddress: models.Address{
				FirstName: "John",
				LastName:  "Doe",
				Address1:  "123 Main St",
				City:      "Anytown",
				Province:  "CA",
				Country:   "US",
				Zip:       "12345",
			},
			BillingAddress: models.Address{
				FirstName: "John",
				LastName:  "Doe",
				Address1:  "123 Main St",
				City:      "Anytown",
				Province:  "CA",
				Country:   "US",
				Zip:       "12345",
			},
		}

		_, err = orderService.CreateOrder(context.Background(), orderInput)
		require.NoError(t, err)
	}

	// Get first page (2 orders)
	result, err := orderService.ListOrders(context.Background(), 2, "", nil)
	require.NoError(t, err)
	assert.Len(t, result.Orders, 2)
	assert.NotEmpty(t, result.NextCursor)

	// Get next page
	result2, err := orderService.ListOrders(context.Background(), 2, result.NextCursor, nil)
	require.NoError(t, err)
	assert.Len(t, result2.Orders, 2)
}
