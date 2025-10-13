package services

import (
	"context"
	"testing"
	"time"

	"github.com/lemnispace/shop-api/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProductService_CreateProduct(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	product := &models.Product{
		Title:       "Test Product",
		Description: "A test product",
		Status:      "active",
		Price:       29.99,
		SKU:         "TEST-SKU-001",
		Tags:        []string{"test", "demo"},
	}

	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)
	assert.NotEmpty(t, product.ID)
	assert.NotZero(t, product.CreatedAt)
	assert.NotZero(t, product.UpdatedAt)
}

func TestProductService_CreateProduct_WithVariants(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	product := &models.Product{
		Title:       "Test Product with Variants",
		Description: "A test product with multiple variants",
		Status:      "active",
		Price:       29.99,
		Variants: []models.ProductVariant{
			{
				Title:     "Small",
				Price:     29.99,
				SKU:       "TEST-SKU-S",
				Inventory: 100,
			},
			{
				Title:     "Medium",
				Price:     34.99,
				SKU:       "TEST-SKU-M",
				Inventory: 75,
			},
			{
				Title:     "Large",
				Price:     39.99,
				SKU:       "TEST-SKU-L",
				Inventory: 50,
			},
		},
	}

	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)
	assert.NotEmpty(t, product.ID)
	assert.Len(t, product.Variants, 3)

	// Verify all variants have IDs
	for i, variant := range product.Variants {
		assert.NotEmpty(t, variant.ID, "Variant %d should have an ID", i)
		assert.Equal(t, product.ID, variant.ProductID)
		assert.Equal(t, product.Title, variant.ProductTitle)
	}
}

func TestProductService_GetProduct(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create a product
	product := &models.Product{
		Title:       "Test Product",
		Description: "A test product",
		Status:      "active",
		Price:       29.99,
		SKU:         "TEST-SKU-001",
	}

	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Get the product
	retrieved, err := productService.GetProduct(context.Background(), product.ID)
	require.NoError(t, err)
	assert.Equal(t, product.ID, retrieved.ID)
	assert.Equal(t, product.Title, retrieved.Title)
	assert.Equal(t, product.Description, retrieved.Description)
	assert.Equal(t, product.Status, retrieved.Status)
	assert.Equal(t, product.Price, retrieved.Price)
	assert.Equal(t, product.SKU, retrieved.SKU)
}

func TestProductService_GetProduct_NotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	_, err := productService.GetProduct(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Equal(t, ErrProductNotFound, err)
}

func TestProductService_GetProduct_EmptyID(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	_, err := productService.GetProduct(context.Background(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "product ID cannot be empty")
}

func TestProductService_UpdateProduct(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create a product
	product := &models.Product{
		Title:       "Test Product",
		Description: "Original description",
		Status:      "active",
		Price:       29.99,
	}

	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Update the product
	originalCreatedAt := product.CreatedAt
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	product.Title = "Updated Product"
	product.Description = "Updated description"
	product.Price = 39.99
	product.Status = "draft"

	err = productService.UpdateProduct(context.Background(), product)
	require.NoError(t, err)

	// Verify update
	updated, err := productService.GetProduct(context.Background(), product.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Product", updated.Title)
	assert.Equal(t, "Updated description", updated.Description)
	assert.Equal(t, 39.99, updated.Price)
	assert.Equal(t, "draft", updated.Status)
	assert.Equal(t, originalCreatedAt.Unix(), updated.CreatedAt.Unix())
	assert.True(t, updated.UpdatedAt.After(originalCreatedAt))
}

func TestProductService_UpdateProduct_NotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	product := &models.Product{
		ID:          "nonexistent",
		Title:       "Test Product",
		Description: "Test",
		Status:      "active",
	}

	err := productService.UpdateProduct(context.Background(), product)
	assert.Error(t, err)
}

func TestProductService_UpdateProduct_EmptyID(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	product := &models.Product{
		Title:       "Test Product",
		Description: "Test",
		Status:      "active",
	}

	err := productService.UpdateProduct(context.Background(), product)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "product ID cannot be empty")
}

func TestProductService_DeleteProduct(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create a product
	product := &models.Product{
		Title:       "Test Product",
		Description: "A test product",
		Status:      "active",
	}

	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Delete the product
	err = productService.DeleteProduct(context.Background(), product.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = productService.GetProduct(context.Background(), product.ID)
	assert.Error(t, err)
	assert.Equal(t, ErrProductNotFound, err)
}

func TestProductService_DeleteProduct_NotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	err := productService.DeleteProduct(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Equal(t, ErrProductNotFound, err)
}

func TestProductService_DeleteProduct_EmptyID(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	err := productService.DeleteProduct(context.Background(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "product ID cannot be empty")
}

func TestProductService_ListProducts(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create multiple products
	products := []*models.Product{
		{
			Title:  "Product A",
			Status: "active",
			Price:  10.00,
			Tags:   []string{"tag1", "tag2"},
		},
		{
			Title:  "Product B",
			Status: "active",
			Price:  20.00,
			Tags:   []string{"tag2", "tag3"},
		},
		{
			Title:  "Product C",
			Status: "draft",
			Price:  30.00,
			Tags:   []string{"tag1", "tag3"},
		},
		{
			Title:  "Product D",
			Status: "active",
			Price:  40.00,
			Tags:   []string{"tag1"},
		},
		{
			Title:  "Product E",
			Status: "archived",
			Price:  50.00,
			Tags:   []string{"tag2"},
		},
	}

	for _, p := range products {
		err := productService.CreateProduct(context.Background(), p)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// List all products
	result, err := productService.ListProducts(context.Background(), 10, "", nil, "", "")
	require.NoError(t, err)
	assert.Len(t, result.Products, 5)
}

func TestProductService_ListProducts_WithStatusFilter(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create multiple products with different statuses
	products := []*models.Product{
		{Title: "Active 1", Status: "active", Price: 10.00},
		{Title: "Active 2", Status: "active", Price: 20.00},
		{Title: "Draft 1", Status: "draft", Price: 30.00},
		{Title: "Archived 1", Status: "archived", Price: 40.00},
	}

	for _, p := range products {
		err := productService.CreateProduct(context.Background(), p)
		require.NoError(t, err)
	}

	// Filter by status
	filters := map[string]interface{}{
		"status": "active",
	}

	result, err := productService.ListProducts(context.Background(), 10, "", filters, "", "")
	require.NoError(t, err)
	assert.Len(t, result.Products, 2)

	// Verify all returned products are active
	for _, product := range result.Products {
		assert.Equal(t, "active", product.Status)
	}
}

func TestProductService_ListProducts_WithPriceFilter(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create multiple products with different prices
	products := []*models.Product{
		{Title: "Cheap", Status: "active", Price: 10.00},
		{Title: "Medium", Status: "active", Price: 25.00},
		{Title: "Expensive", Status: "active", Price: 50.00},
	}

	for _, p := range products {
		err := productService.CreateProduct(context.Background(), p)
		require.NoError(t, err)
	}

	// Filter by price range
	filters := map[string]interface{}{
		"price_min": "20",
		"price_max": "40",
	}

	result, err := productService.ListProducts(context.Background(), 10, "", filters, "", "")
	require.NoError(t, err)
	assert.Len(t, result.Products, 1)
	assert.Equal(t, "Medium", result.Products[0].Title)
}

func TestProductService_ListProducts_WithTagFilter(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create multiple products with different tags
	products := []*models.Product{
		{Title: "Product 1", Status: "active", Tags: []string{"electronics", "featured"}},
		{Title: "Product 2", Status: "active", Tags: []string{"clothing", "sale"}},
		{Title: "Product 3", Status: "active", Tags: []string{"electronics", "sale"}},
	}

	for _, p := range products {
		err := productService.CreateProduct(context.Background(), p)
		require.NoError(t, err)
	}

	// Filter by tag
	filters := map[string]interface{}{
		"tag": "electronics",
	}

	result, err := productService.ListProducts(context.Background(), 10, "", filters, "", "")
	require.NoError(t, err)
	assert.Len(t, result.Products, 2)

	// Verify all returned products have the electronics tag
	for _, product := range result.Products {
		assert.Contains(t, product.Tags, "electronics")
	}
}

func TestProductService_ListProducts_Pagination(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create 5 products
	for i := 0; i < 5; i++ {
		product := &models.Product{
			Title:  "Product " + string(rune('A'+i)),
			Status: "active",
			Price:  float64(10 + i*10),
		}
		err := productService.CreateProduct(context.Background(), product)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Get first page (2 products)
	result, err := productService.ListProducts(context.Background(), 2, "", nil, "", "")
	require.NoError(t, err)
	assert.Len(t, result.Products, 2)
	assert.NotEmpty(t, result.NextCursor)

	// Get next page
	result2, err := productService.ListProducts(context.Background(), 2, result.NextCursor, nil, "", "")
	require.NoError(t, err)
	assert.Len(t, result2.Products, 2)
}

func TestProductService_ListProducts_SortByTitle(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create products in random order
	products := []*models.Product{
		{Title: "Zebra Product", Status: "active", Price: 10.00},
		{Title: "Apple Product", Status: "active", Price: 20.00},
		{Title: "Mango Product", Status: "active", Price: 30.00},
	}

	for _, p := range products {
		err := productService.CreateProduct(context.Background(), p)
		require.NoError(t, err)
	}

	// Sort by title ascending
	result, err := productService.ListProducts(context.Background(), 10, "", nil, "title", "asc")
	require.NoError(t, err)
	assert.Len(t, result.Products, 3)
	assert.Equal(t, "Apple Product", result.Products[0].Title)
	assert.Equal(t, "Mango Product", result.Products[1].Title)
	assert.Equal(t, "Zebra Product", result.Products[2].Title)
}

func TestProductService_ListProducts_SortByPrice(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create products with different prices
	products := []*models.Product{
		{Title: "Product 1", Status: "active", Price: 50.00},
		{Title: "Product 2", Status: "active", Price: 10.00},
		{Title: "Product 3", Status: "active", Price: 30.00},
	}

	for _, p := range products {
		err := productService.CreateProduct(context.Background(), p)
		require.NoError(t, err)
	}

	// Sort by price descending
	result, err := productService.ListProducts(context.Background(), 10, "", nil, "price", "desc")
	require.NoError(t, err)
	assert.Len(t, result.Products, 3)
	assert.Equal(t, 50.00, result.Products[0].Price)
	assert.Equal(t, 30.00, result.Products[1].Price)
	assert.Equal(t, 10.00, result.Products[2].Price)
}

func TestProductService_CountProducts(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create multiple products
	for i := 0; i < 7; i++ {
		product := &models.Product{
			Title:  "Product " + string(rune('A'+i)),
			Status: "active",
		}
		err := productService.CreateProduct(context.Background(), product)
		require.NoError(t, err)
	}

	// Count products
	count, err := productService.CountProducts(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, 7, count)
}

func TestProductService_AddProductVariant(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create a product
	product := &models.Product{
		Title:  "Test Product",
		Status: "active",
	}

	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Add a variant
	variant := &models.ProductVariant{
		Title:     "New Variant",
		Price:     29.99,
		SKU:       "NEW-VAR-001",
		Inventory: 100,
	}

	err = productService.AddProductVariant(context.Background(), product.ID, variant)
	require.NoError(t, err)
	assert.NotEmpty(t, variant.ID)
	assert.Equal(t, product.ID, variant.ProductID)
	assert.Equal(t, product.Title, variant.ProductTitle)

	// Verify variant was added
	retrieved, err := productService.GetProduct(context.Background(), product.ID)
	require.NoError(t, err)
	assert.Len(t, retrieved.Variants, 1)
	assert.Equal(t, variant.Title, retrieved.Variants[0].Title)
}

func TestProductService_AddProductVariant_ProductNotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	variant := &models.ProductVariant{
		Title: "New Variant",
		Price: 29.99,
	}

	err := productService.AddProductVariant(context.Background(), "nonexistent", variant)
	assert.Error(t, err)
}

func TestProductService_UpdateProductVariant(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create a product with a variant
	product := &models.Product{
		Title:  "Test Product",
		Status: "active",
		Variants: []models.ProductVariant{
			{
				Title:     "Original Variant",
				Price:     29.99,
				SKU:       "ORIG-001",
				Inventory: 100,
			},
		},
	}

	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Update the variant
	variantID := product.Variants[0].ID
	updatedVariant := &models.ProductVariant{
		ID:        variantID,
		Title:     "Updated Variant",
		Price:     39.99,
		SKU:       "UPDATED-001",
		Inventory: 75,
	}

	err = productService.UpdateProductVariant(context.Background(), product.ID, updatedVariant)
	require.NoError(t, err)

	// Verify update
	retrieved, err := productService.GetProduct(context.Background(), product.ID)
	require.NoError(t, err)
	assert.Len(t, retrieved.Variants, 1)
	assert.Equal(t, "Updated Variant", retrieved.Variants[0].Title)
	assert.Equal(t, 39.99, retrieved.Variants[0].Price)
	assert.Equal(t, "UPDATED-001", retrieved.Variants[0].SKU)
	assert.Equal(t, 75, retrieved.Variants[0].Inventory)
}

func TestProductService_UpdateProductVariant_NotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create a product without variants
	product := &models.Product{
		Title:  "Test Product",
		Status: "active",
	}

	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Try to update a non-existent variant
	variant := &models.ProductVariant{
		ID:    "nonexistent",
		Title: "Updated Variant",
		Price: 39.99,
	}

	err = productService.UpdateProductVariant(context.Background(), product.ID, variant)
	assert.Error(t, err)
	assert.Equal(t, ErrVariantNotFound, err)
}

func TestProductService_DeleteProductVariant(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create a product with two variants
	product := &models.Product{
		Title:  "Test Product",
		Status: "active",
		Variants: []models.ProductVariant{
			{
				Title:     "Variant 1",
				Price:     29.99,
				SKU:       "VAR-001",
				Inventory: 100,
			},
			{
				Title:     "Variant 2",
				Price:     39.99,
				SKU:       "VAR-002",
				Inventory: 50,
			},
		},
	}

	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Delete the first variant
	variantID := product.Variants[0].ID
	err = productService.DeleteProductVariant(context.Background(), product.ID, variantID)
	require.NoError(t, err)

	// Verify deletion
	retrieved, err := productService.GetProduct(context.Background(), product.ID)
	require.NoError(t, err)
	assert.Len(t, retrieved.Variants, 1)
	assert.Equal(t, "Variant 2", retrieved.Variants[0].Title)
}

func TestProductService_DeleteProductVariant_NotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create a product without variants
	product := &models.Product{
		Title:  "Test Product",
		Status: "active",
	}

	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Try to delete a non-existent variant
	err = productService.DeleteProductVariant(context.Background(), product.ID, "nonexistent")
	assert.Error(t, err)
	assert.Equal(t, ErrVariantNotFound, err)
}

func TestProductService_AddProductImage(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create a product
	product := &models.Product{
		Title:  "Test Product",
		Status: "active",
	}

	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Add an image
	image := &models.Image{
		URL:      "https://example.com/image.jpg",
		AltText:  "Test image",
		Width:    800,
		Height:   600,
		Position: 1,
	}

	err = productService.AddProductImage(context.Background(), product.ID, image)
	require.NoError(t, err)
	assert.NotEmpty(t, image.ID)
	assert.NotZero(t, image.CreatedAt)
	assert.NotZero(t, image.UpdatedAt)

	// Verify image was added
	retrieved, err := productService.GetProduct(context.Background(), product.ID)
	require.NoError(t, err)
	assert.Len(t, retrieved.Images, 1)
	assert.Equal(t, image.URL, retrieved.Images[0].URL)
	assert.Equal(t, image.AltText, retrieved.Images[0].AltText)
}

func TestProductService_AddProductImage_ProductNotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	image := &models.Image{
		URL:     "https://example.com/image.jpg",
		AltText: "Test image",
	}

	err := productService.AddProductImage(context.Background(), "nonexistent", image)
	assert.Error(t, err)
}

func TestProductService_AssociateImageWithVariant(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create a product with a variant and an image
	product := &models.Product{
		Title:  "Test Product",
		Status: "active",
		Variants: []models.ProductVariant{
			{
				Title:     "Test Variant",
				Price:     29.99,
				Inventory: 100,
			},
		},
		Images: []models.Image{
			{
				URL:     "https://example.com/image.jpg",
				AltText: "Test image",
			},
		},
	}

	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	variantID := product.Variants[0].ID
	imageID := product.Images[0].ID

	// Associate image with variant
	err = productService.AssociateImageWithVariant(context.Background(), product.ID, variantID, imageID)
	require.NoError(t, err)

	// Verify association
	retrieved, err := productService.GetProduct(context.Background(), product.ID)
	require.NoError(t, err)
	assert.Len(t, retrieved.Images, 1)
	assert.Contains(t, retrieved.Images[0].Variants, variantID)
}

func TestProductService_AssociateImageWithVariant_VariantNotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create a product with an image but no variants
	product := &models.Product{
		Title:  "Test Product",
		Status: "active",
		Images: []models.Image{
			{
				URL:     "https://example.com/image.jpg",
				AltText: "Test image",
			},
		},
	}

	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	imageID := product.Images[0].ID

	// Try to associate with non-existent variant
	err = productService.AssociateImageWithVariant(context.Background(), product.ID, "nonexistent", imageID)
	assert.Error(t, err)
	assert.Equal(t, ErrVariantNotFound, err)
}

func TestProductService_AssociateImageWithVariant_ImageNotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create a product with a variant but no images
	product := &models.Product{
		Title:  "Test Product",
		Status: "active",
		Variants: []models.ProductVariant{
			{
				Title:     "Test Variant",
				Price:     29.99,
				Inventory: 100,
			},
		},
	}

	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	variantID := product.Variants[0].ID

	// Try to associate with non-existent image
	err = productService.AssociateImageWithVariant(context.Background(), product.ID, variantID, "nonexistent")
	assert.Error(t, err)
	assert.Equal(t, ErrImageNotFound, err)
}

func TestProductService_ListProductVariants(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create a product with multiple variants
	product := &models.Product{
		Title:  "Test Product",
		Status: "active",
		Variants: []models.ProductVariant{
			{Title: "Variant 1", Price: 10.00},
			{Title: "Variant 2", Price: 20.00},
			{Title: "Variant 3", Price: 30.00},
		},
	}

	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// List variants
	variants, nextCursor, err := productService.ListProductVariants(context.Background(), product.ID, 10, "")
	require.NoError(t, err)
	assert.Len(t, variants, 3)
	assert.Empty(t, nextCursor)
}

func TestProductService_ListProductVariants_Pagination(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create a product with 5 variants
	variants := []models.ProductVariant{}
	for i := 0; i < 5; i++ {
		variants = append(variants, models.ProductVariant{
			Title: "Variant " + string(rune('A'+i)),
			Price: float64(10 + i*10),
		})
	}

	product := &models.Product{
		Title:    "Test Product",
		Status:   "active",
		Variants: variants,
	}

	err := productService.CreateProduct(context.Background(), product)
	require.NoError(t, err)

	// Get first page (2 variants)
	page1, cursor1, err := productService.ListProductVariants(context.Background(), product.ID, 2, "")
	require.NoError(t, err)
	assert.Len(t, page1, 2)
	assert.NotEmpty(t, cursor1)

	// Get second page
	page2, cursor2, err := productService.ListProductVariants(context.Background(), product.ID, 2, cursor1)
	require.NoError(t, err)
	assert.Len(t, page2, 2)
	assert.NotEmpty(t, cursor2)

	// Get third page
	page3, cursor3, err := productService.ListProductVariants(context.Background(), product.ID, 2, cursor2)
	require.NoError(t, err)
	assert.Len(t, page3, 1)
	assert.Empty(t, cursor3)
}

func TestProductService_ListAllVariants(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	productService := NewProductService(client, tableName)

	// Create multiple products with variants
	for i := 0; i < 3; i++ {
		product := &models.Product{
			Title:  "Product " + string(rune('A'+i)),
			Status: "active",
			Variants: []models.ProductVariant{
				{Title: "Variant 1", Price: float64(10 + i*10)},
				{Title: "Variant 2", Price: float64(20 + i*10)},
			},
		}
		err := productService.CreateProduct(context.Background(), product)
		require.NoError(t, err)
	}

	// List all variants
	variants, _, err := productService.ListAllVariants(context.Background(), 100, "", nil, "", "")
	require.NoError(t, err)
	assert.Len(t, variants, 6) // 3 products * 2 variants each
}
