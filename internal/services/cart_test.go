package services

import (
	"context"
	"testing"
	"time"

	"github.com/lemnispace/shop-api/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCartService_CreateCart(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)
	assert.NotEmpty(t, cart.ID)
	assert.Equal(t, "cust_123", cart.CustomerID)
	assert.NotZero(t, cart.CreatedAt)
	assert.NotZero(t, cart.UpdatedAt)
	assert.NotZero(t, cart.ExpiresAt)
	assert.Empty(t, cart.Items)
	assert.Equal(t, int64(0), cart.Subtotal)
	assert.Equal(t, int64(0), cart.EstimatedTax)
	assert.Equal(t, int64(0), cart.TotalPrice)
}

func TestCartService_CreateCart_WithoutCustomer(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	cart, err := cartService.CreateCart(context.Background(), "")
	require.NoError(t, err)
	assert.NotEmpty(t, cart.ID)
	assert.Empty(t, cart.CustomerID)
	assert.NotZero(t, cart.CreatedAt)
}

func TestCartService_GetCart(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Create a cart
	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	// Get the cart
	retrieved, err := cartService.GetCart(context.Background(), cart.ID)
	require.NoError(t, err)
	assert.Equal(t, cart.ID, retrieved.ID)
	assert.Equal(t, cart.CustomerID, retrieved.CustomerID)
}

func TestCartService_GetCart_NotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	_, err := cartService.GetCart(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Equal(t, ErrCartNotFound, err)
}

func TestCartService_AddItem(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Create a product
	product := &models.Product{
		Title:       "Test Product",
		Description: "Test",
		Status:      "active",
		Price:       2999, // $29.99 in cents
		Inventory:   100,
	}
	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Create a cart
	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	// Add item to cart
	cartItem, err := cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
		ProductID: product.ID,
		Quantity:  2,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, cartItem.ID)
	assert.Equal(t, product.ID, cartItem.ProductID)
	assert.Equal(t, 2, cartItem.Quantity)
	assert.Equal(t, int64(2999), cartItem.Price)
	assert.NotNil(t, cartItem.Product)
	assert.Equal(t, product.Title, cartItem.Product.Title)

	// Verify cart totals are updated
	updatedCart, err := cartService.GetCart(context.Background(), cart.ID)
	require.NoError(t, err)
	assert.Len(t, updatedCart.Items, 1)
	assert.Equal(t, int64(5998), updatedCart.Subtotal) // 2999 * 2 cents
	assert.Greater(t, updatedCart.EstimatedTax, int64(0))
	assert.Greater(t, updatedCart.EstimatedShipping, int64(0))
	assert.Greater(t, updatedCart.TotalPrice, updatedCart.Subtotal)
}

func TestCartService_AddItem_WithVariant(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Create a product with variants
	product := &models.Product{
		Title:       "Test Product",
		Description: "Test",
		Status:      "active",
		Variants: []models.ProductVariant{
			{
				Title:     "Small",
				Price:     1999, // $19.99 in cents
				SKU:       "TEST-S",
				Inventory: 50,
			},
			{
				Title:     "Large",
				Price:     2999, // $29.99 in cents
				SKU:       "TEST-L",
				Inventory: 30,
			},
		},
	}
	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Create a cart
	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	// Add item with specific variant
	cartItem, err := cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
		ProductID: product.ID,
		VariantID: product.Variants[1].ID, // Large variant
		Quantity:  1,
	})
	require.NoError(t, err)
	assert.Equal(t, product.Variants[1].ID, cartItem.VariantID)
	assert.Equal(t, int64(2999), cartItem.Price) // Large variant price
	assert.NotNil(t, cartItem.Variant)
	assert.Equal(t, "Large", cartItem.Variant.Title)
}

func TestCartService_AddItem_InsufficientInventory(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Create a product with limited inventory
	product := &models.Product{
		Title:       "Test Product",
		Description: "Test",
		Status:      "active",
		Price:       2999, // $29.99 in cents
		Inventory:   5,
	}
	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Create a cart
	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	// Try to add more items than available
	_, err = cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
		ProductID: product.ID,
		Quantity:  10, // More than inventory
	})
	assert.Error(t, err)
	assert.Equal(t, ErrProductNotInStock, err)
}

func TestCartService_AddItem_VariantNotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Create a product
	product := &models.Product{
		Title:       "Test Product",
		Description: "Test",
		Status:      "active",
		Price:       2999, // $29.99 in cents
		Inventory:   100,
	}
	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Create a cart
	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	// Try to add item with non-existent variant
	_, err = cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
		ProductID: product.ID,
		VariantID: "nonexistent",
		Quantity:  1,
	})
	assert.Error(t, err)
	assert.Equal(t, ErrVariantNotFound, err)
}

func TestCartService_AddItem_ProductNotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Create a cart
	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	// Try to add non-existent product
	_, err = cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
		ProductID: "nonexistent",
		Quantity:  1,
	})
	assert.Error(t, err)
}

func TestCartService_AddItem_CartNotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Try to add item to non-existent cart
	_, err := cartService.AddItem(context.Background(), "nonexistent", models.CartItemInput{
		ProductID: "prod_123",
		Quantity:  1,
	})
	assert.Error(t, err)
	assert.Equal(t, ErrCartNotFound, err)
}

func TestCartService_AddItem_MultipleItems(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Create products
	product1 := &models.Product{
		Title:     "Product 1",
		Status:    "active",
		Price:     1000, // $10.00 in cents
		Inventory: 100,
	}
	product2 := &models.Product{
		Title:     "Product 2",
		Status:    "active",
		Price:     2000, // $20.00 in cents
		Inventory: 100,
	}
	err := productService.CreateProduct(context.Background(), product1)
	require.NoError(t, err)
	err = productService.CreateProduct(context.Background(), product2)
	require.NoError(t, err)

	// Create a cart
	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	// Add first item
	_, err = cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
		ProductID: product1.ID,
		Quantity:  2,
	})
	require.NoError(t, err)

	// Add second item
	_, err = cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
		ProductID: product2.ID,
		Quantity:  1,
	})
	require.NoError(t, err)

	// Verify cart has both items
	updatedCart, err := cartService.GetCart(context.Background(), cart.ID)
	require.NoError(t, err)
	assert.Len(t, updatedCart.Items, 2)
	assert.Equal(t, int64(4000), updatedCart.Subtotal) // (1000 * 2) + (2000 * 1) in cents
}

func TestCartService_UpdateItem(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Create a product
	product := &models.Product{
		Title:       "Test Product",
		Description: "Test",
		Status:      "active",
		Price:       2999, // $29.99 in cents
		Inventory:   100,
	}
	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Create a cart and add item
	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	cartItem, err := cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
		ProductID: product.ID,
		Quantity:  2,
	})
	require.NoError(t, err)

	// Update item quantity
	updatedItem, err := cartService.UpdateItem(context.Background(), cart.ID, cartItem.ID, 5)
	require.NoError(t, err)
	assert.Equal(t, 5, updatedItem.Quantity)

	// Verify cart totals are updated
	updatedCart, err := cartService.GetCart(context.Background(), cart.ID)
	require.NoError(t, err)
	assert.Equal(t, 5, updatedCart.Items[0].Quantity)
	assert.Equal(t, int64(14995), updatedCart.Subtotal) // 2999 * 5 cents
}

func TestCartService_UpdateItem_InsufficientInventory(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Create a product with limited inventory
	product := &models.Product{
		Title:       "Test Product",
		Description: "Test",
		Status:      "active",
		Price:       2999, // $29.99 in cents
		Inventory:   10,
	}
	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Create a cart and add item
	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	cartItem, err := cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
		ProductID: product.ID,
		Quantity:  2,
	})
	require.NoError(t, err)

	// Try to update to more than available inventory
	_, err = cartService.UpdateItem(context.Background(), cart.ID, cartItem.ID, 20)
	assert.Error(t, err)
	assert.Equal(t, ErrProductNotInStock, err)
}

func TestCartService_UpdateItem_ItemNotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Create a cart
	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	// Try to update non-existent item
	_, err = cartService.UpdateItem(context.Background(), cart.ID, "nonexistent", 5)
	assert.Error(t, err)
	assert.Equal(t, ErrCartItemNotFound, err)
}

func TestCartService_UpdateItem_CartNotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Try to update item in non-existent cart
	_, err := cartService.UpdateItem(context.Background(), "nonexistent", "item_123", 5)
	assert.Error(t, err)
	assert.Equal(t, ErrCartNotFound, err)
}

func TestCartService_RemoveItem(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Create a product
	product := &models.Product{
		Title:       "Test Product",
		Description: "Test",
		Status:      "active",
		Price:       2999, // $29.99 in cents
		Inventory:   100,
	}
	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Create a cart and add item
	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	cartItem, err := cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
		ProductID: product.ID,
		Quantity:  2,
	})
	require.NoError(t, err)

	// Remove the item
	err = cartService.RemoveItem(context.Background(), cart.ID, cartItem.ID)
	require.NoError(t, err)

	// Verify item was removed
	updatedCart, err := cartService.GetCart(context.Background(), cart.ID)
	require.NoError(t, err)
	assert.Empty(t, updatedCart.Items)
	assert.Equal(t, int64(0), updatedCart.Subtotal)
}

func TestCartService_RemoveItem_ItemNotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Create a cart
	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	// Try to remove non-existent item
	err = cartService.RemoveItem(context.Background(), cart.ID, "nonexistent")
	assert.Error(t, err)
	assert.Equal(t, ErrCartItemNotFound, err)
}

func TestCartService_RemoveItem_CartNotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Try to remove item from non-existent cart
	err := cartService.RemoveItem(context.Background(), "nonexistent", "item_123")
	assert.Error(t, err)
	assert.Equal(t, ErrCartNotFound, err)
}

func TestCartService_RemoveItem_MultipleItems(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Create products
	product1 := &models.Product{
		Title:     "Product 1",
		Status:    "active",
		Price:     1000, // $10.00 in cents
		Inventory: 100,
	}
	product2 := &models.Product{
		Title:     "Product 2",
		Status:    "active",
		Price:     2000, // $20.00 in cents
		Inventory: 100,
	}
	err := productService.CreateProduct(context.Background(), product1)
	require.NoError(t, err)
	err = productService.CreateProduct(context.Background(), product2)
	require.NoError(t, err)

	// Create a cart and add both items
	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	item1, err := cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
		ProductID: product1.ID,
		Quantity:  2,
	})
	require.NoError(t, err)

	_, err = cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
		ProductID: product2.ID,
		Quantity:  1,
	})
	require.NoError(t, err)

	// Remove first item
	err = cartService.RemoveItem(context.Background(), cart.ID, item1.ID)
	require.NoError(t, err)

	// Verify only second item remains
	updatedCart, err := cartService.GetCart(context.Background(), cart.ID)
	require.NoError(t, err)
	assert.Len(t, updatedCart.Items, 1)
	assert.Equal(t, product2.ID, updatedCart.Items[0].ProductID)
	assert.Equal(t, int64(2000), updatedCart.Subtotal) // $20.00 in cents
}

func TestCartService_GetCheckoutURL(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Create a cart
	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	// Get checkout URL
	response, err := cartService.GetCheckoutURL(context.Background(), cart.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, response.CheckoutURL)
	assert.Contains(t, response.CheckoutURL, cart.ID)
}

func TestCartService_GetCheckoutURL_CartNotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Try to get checkout URL for non-existent cart
	_, err := cartService.GetCheckoutURL(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Equal(t, ErrCartNotFound, err)
}

func TestCartService_GetCartsByCustomer(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	customerID := "cust_123"

	// Create multiple carts for the same customer
	cart1, err := cartService.CreateCart(context.Background(), customerID)
	require.NoError(t, err)

	cart2, err := cartService.CreateCart(context.Background(), customerID)
	require.NoError(t, err)

	// Create cart for different customer
	_, err = cartService.CreateCart(context.Background(), "cust_456")
	require.NoError(t, err)

	// Get carts for customer
	carts, err := cartService.GetCartsByCustomer(context.Background(), customerID, false)
	require.NoError(t, err)
	assert.Len(t, carts, 2)

	// Verify returned carts belong to the customer
	cartIDs := []string{cart1.ID, cart2.ID}
	for _, cart := range carts {
		assert.Equal(t, customerID, cart.CustomerID)
		assert.Contains(t, cartIDs, cart.ID)
	}
}

func TestCartService_GetCartsByCustomer_NoResults(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Get carts for customer with no carts
	carts, err := cartService.GetCartsByCustomer(context.Background(), "cust_nonexistent", false)
	require.NoError(t, err)
	assert.Empty(t, carts)
}

func TestCartService_PriceCalculations(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Create a product
	product := &models.Product{
		Title:       "Test Product",
		Description: "Test",
		Status:      "active",
		Price:       10000, // $100.00 in cents
		Inventory:   100,
	}
	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Create a cart and add item
	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	_, err = cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
		ProductID: product.ID,
		Quantity:  2,
	})
	require.NoError(t, err)

	// Verify price calculations
	updatedCart, err := cartService.GetCart(context.Background(), cart.ID)
	require.NoError(t, err)

	expectedSubtotal := int64(20000) // 10000 * 2 cents
	expectedTax := int64(1800)       // (20000 * 900) / 10000 = 1800 cents
	expectedShipping := int64(599)   // $5.99 in cents
	expectedTotal := expectedSubtotal + expectedTax + expectedShipping

	assert.Equal(t, expectedSubtotal, updatedCart.Subtotal)
	assert.Equal(t, expectedTax, updatedCart.EstimatedTax)
	assert.Equal(t, expectedShipping, updatedCart.EstimatedShipping)
	assert.Equal(t, expectedTotal, updatedCart.TotalPrice)
}

func TestCartService_CartExpiration(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Create a cart
	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	// Verify ExpiresAt is set to 24 hours from now
	expectedExpiry := time.Now().Add(24 * time.Hour)
	assert.WithinDuration(t, expectedExpiry, cart.ExpiresAt, 5*time.Second)
}

func TestCartService_CartWithCustomization(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)
	cartService := NewCartService(client, productService, tableName)

	// Create a product
	product := &models.Product{
		Title:       "Customizable Product",
		Description: "Test",
		Status:      "active",
		Price:       2999, // $29.99 in cents
		Inventory:   100,
	}
	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Create a cart
	cart, err := cartService.CreateCart(context.Background(), "cust_123")
	require.NoError(t, err)

	// Add item with customization data
	customizationData := map[string]interface{}{
		"imageURL": "https://example.com/custom-image.jpg",
		"text":     "Custom Text",
		"color":    "blue",
	}

	cartItem, err := cartService.AddItem(context.Background(), cart.ID, models.CartItemInput{
		ProductID:          product.ID,
		Quantity:           1,
		CustomizationData: customizationData,
	})
	require.NoError(t, err)
	assert.NotNil(t, cartItem.CustomizationData)
	assert.Equal(t, customizationData["imageURL"], cartItem.CustomizationData["imageURL"])
	assert.Equal(t, customizationData["text"], cartItem.CustomizationData["text"])

	// Verify customization data is persisted
	updatedCart, err := cartService.GetCart(context.Background(), cart.ID)
	require.NoError(t, err)
	assert.NotNil(t, updatedCart.Items[0].CustomizationData)
	assert.Equal(t, customizationData["imageURL"], updatedCart.Items[0].CustomizationData["imageURL"])
}
