package unit

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/internal/services"
	"github.com/lemnispace/shop-api/tests"
	"github.com/stretchr/testify/assert"
)

// TestMain runs before and after all tests in the package
func TestMain(m *testing.M) {
	// Run all the tests
	exitCode := m.Run()

	// No need to call FinalCleanup() here as it will be called by the api package's TestMain

	// Exit with the same code from the tests
	os.Exit(exitCode)
}

// TestDynamoDBProductService tests the basic functionality of the DynamoDBProductService
func TestDynamoDBProductService(t *testing.T) {
	// Create a new DynamoDB product service for testing
	productService, client, err := setupProductServiceTest()
	if err != nil {
		t.Fatalf("Error creating test product service: %v", err)
	}

	// Cleanup test data after test is done
	defer cleanupProductTest(t, client)

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
	assert.NoError(t, err, "Error creating product")

	// Verify the product was created (ID should be set)
	assert.NotEmpty(t, testProduct.ID, "Expected product ID to be set after creation")

	// Get the product back from the service
	retrievedProduct, err := productService.GetProduct(context.Background(), testProduct.ID)
	assert.NoError(t, err, "Error retrieving product")

	// Verify the retrieved product matches what we created
	assert.Equal(t, testProduct.Title, retrievedProduct.Title, "Product title mismatch")
	assert.Equal(t, testProduct.Price, retrievedProduct.Price, "Product price mismatch")
	assert.Equal(t, testProduct.ID, retrievedProduct.ID, "Product ID mismatch")

	// Success - we've verified we can create and retrieve a product
	t.Logf("Successfully created and retrieved product with ID: %s", testProduct.ID)
}

// TestProductSingleTableDesign tests that the product service properly uses the single table design
func TestProductSingleTableDesign(t *testing.T) {
	productID := "prod123"

	// Test that the ProductKey function creates correct keys
	pk, sk := services.ProductKey(productID)

	// Verify the format matches our single table design
	expectedPK := fmt.Sprintf("%s#%s", services.EntityProduct, productID)
	expectedSK := fmt.Sprintf("%s#%s", services.EntityProduct, productID)

	assert.Equal(t, expectedPK, pk, "Product PK format doesn't match expected pattern")
	assert.Equal(t, expectedSK, sk, "Product SK format doesn't match expected pattern")
}

// Helper function to set up product service test
func setupProductServiceTest() (services.ProductService, *dynamodb.Client, error) {
	client, err := tests.SetupDynamoDBForTesting()
	if err != nil {
		return nil, nil, err
	}

	productService := services.NewProductService(client, tests.TestTableName())
	return productService, client, nil
}

// Helper function to clean up after tests
func cleanupProductTest(t *testing.T, client *dynamodb.Client) {
	if client != nil {
		err := tests.TeardownDynamoDBForTesting(client)
		if err != nil {
			t.Logf("Error cleaning up test data: %v", err)
		}
	}
}
