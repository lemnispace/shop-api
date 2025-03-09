package unit

import (
	"context"
	"testing"

	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/tests"
)

// TestDynamoDBProductService tests the basic functionality of the DynamoDBProductService
func TestDynamoDBProductService(t *testing.T) {
	// Create a new DynamoDB product service for testing
	productService, err := tests.CreateTestProductService()
	if err != nil {
		t.Fatalf("Error creating test product service: %v", err)
	}

	// Create a test product
	testProduct := &models.Product{
		Title:       "Test Product",
		Description: "A product for unit testing",
		Price:       19.99,
		SKU:         "TEST-UNIT-001",
		Status:      "active",
		Inventory:   50,
	}

	// Create the product in the service
	err = productService.CreateProduct(context.Background(), testProduct)
	if err != nil {
		t.Fatalf("Error creating product: %v", err)
	}

	// Verify the product was created (ID should be set)
	if testProduct.ID == "" {
		t.Error("Expected product ID to be set after creation")
	}

	// Get the product back from the service
	retrievedProduct, err := productService.GetProduct(context.Background(), testProduct.ID)
	if err != nil {
		t.Fatalf("Error retrieving product: %v", err)
	}

	// Verify the retrieved product matches what we created
	if retrievedProduct.Title != testProduct.Title {
		t.Errorf("Expected title %s, got %s", testProduct.Title, retrievedProduct.Title)
	}
	if retrievedProduct.Price != testProduct.Price {
		t.Errorf("Expected price %f, got %f", testProduct.Price, retrievedProduct.Price)
	}

	// Test counting products
	count, err := productService.CountProducts(context.Background(), nil)
	if err != nil {
		t.Fatalf("Error counting products: %v", err)
	}

	// Verify we have at least the product we created
	if count < 1 {
		t.Errorf("Expected at least 1 product, but got %d", count)
	}

	t.Logf("Successfully created and retrieved product with ID: %s", testProduct.ID)
}
