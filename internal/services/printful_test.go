package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lemnispace/shop-api/internal/models"
)

// mockProductService is a mock implementation for testing
type mockProductService struct {
	createProductFunc func(ctx context.Context, product *models.Product) error
}

func (m *mockProductService) CreateProduct(ctx context.Context, product *models.Product) error {
	if m.createProductFunc != nil {
		return m.createProductFunc(ctx, product)
	}
	return nil
}

func (m *mockProductService) GetProduct(ctx context.Context, id string) (*models.Product, error) {
	return nil, nil
}

func (m *mockProductService) UpdateProduct(ctx context.Context, product *models.Product) error {
	return nil
}

func (m *mockProductService) DeleteProduct(ctx context.Context, id string) error {
	return nil
}

func (m *mockProductService) ListProducts(ctx context.Context, limit int, cursor string, filters map[string]interface{}, sortKey, sortOrder string) (*ProductListResult, error) {
	return nil, nil
}

func (m *mockProductService) CountProducts(ctx context.Context, filters map[string]interface{}) (int, error) {
	return 0, nil
}

func (m *mockProductService) ListProductVariants(ctx context.Context, productID string, limit int, cursor string) ([]models.ProductVariant, string, error) {
	return nil, "", nil
}

func (m *mockProductService) ListAllVariants(ctx context.Context, limit int, cursor string, filters map[string]interface{}, sortKey, sortOrder string) ([]models.ProductVariant, string, error) {
	return nil, "", nil
}

func (m *mockProductService) AddProductVariant(ctx context.Context, productID string, variant *models.ProductVariant) error {
	return nil
}

func (m *mockProductService) UpdateProductVariant(ctx context.Context, productID string, variant *models.ProductVariant) error {
	return nil
}

func (m *mockProductService) DeleteProductVariant(ctx context.Context, productID string, variantID string) error {
	return nil
}

func (m *mockProductService) AddProductImage(ctx context.Context, productID string, image *models.Image) error {
	return nil
}

func (m *mockProductService) AssociateImageWithVariant(ctx context.Context, productID string, variantID string, imageID string) error {
	return nil
}

// TestPrintfulClient_GetProducts tests fetching products from Printful
func TestPrintfulClient_GetProducts(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/products" {
			t.Errorf("Expected path /products, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("Expected Authorization header with Bearer token")
		}

		// Return mock response
		response := models.PrintfulAPIResponse{
			Code: 200,
			Result: []models.PrintfulProduct{
				{
					ID:          71,
					Name:        "Canvas Print",
					Variants:    5,
					IsSyncable:  true,
					Description: "High-quality canvas print",
					Category:    "Wall Art",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server
	mockProductSvc := &mockProductService{}
	client := NewPrintfulClient("test-api-key", mockProductSvc)
	client.baseURL = server.URL

	// Test GetProducts
	products, err := client.GetProducts(context.Background())
	if err != nil {
		t.Fatalf("GetProducts failed: %v", err)
	}

	if len(products) != 1 {
		t.Errorf("Expected 1 product, got %d", len(products))
	}

	if products[0].Name != "Canvas Print" {
		t.Errorf("Expected product name 'Canvas Print', got '%s'", products[0].Name)
	}
}

// TestPrintfulClient_GetProduct tests fetching a single product
func TestPrintfulClient_GetProduct(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/products/71" {
			t.Errorf("Expected path /products/71, got %s", r.URL.Path)
		}

		response := models.PrintfulAPIResponse{
			Code: 200,
			Result: map[string]interface{}{
				"product": models.PrintfulProduct{
					ID:          71,
					Name:        "Canvas Print",
					Description: "High-quality canvas print",
					Category:    "Wall Art",
					IsSyncable:  true,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	mockProductSvc := &mockProductService{}
	client := NewPrintfulClient("test-api-key", mockProductSvc)
	client.baseURL = server.URL

	product, err := client.GetProduct(context.Background(), 71)
	if err != nil {
		t.Fatalf("GetProduct failed: %v", err)
	}

	if product.ID != 71 {
		t.Errorf("Expected product ID 71, got %d", product.ID)
	}
	if product.Name != "Canvas Print" {
		t.Errorf("Expected product name 'Canvas Print', got '%s'", product.Name)
	}
}

// TestPrintfulClient_GetProductVariants tests fetching variants
func TestPrintfulClient_GetProductVariants(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := models.PrintfulAPIResponse{
			Code: 200,
			Result: map[string]interface{}{
				"variants": []models.PrintfulVariant{
					{
						ID:        4012,
						ProductID: 71,
						Name:      "12x16 Canvas",
						Size:      "12x16",
						Color:     "White",
						Price:     "19.95",
						InStock:   true,
					},
					{
						ID:        4013,
						ProductID: 71,
						Name:      "16x20 Canvas",
						Size:      "16x20",
						Color:     "White",
						Price:     "24.95",
						InStock:   true,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	mockProductSvc := &mockProductService{}
	client := NewPrintfulClient("test-api-key", mockProductSvc)
	client.baseURL = server.URL

	variants, err := client.GetProductVariants(context.Background(), 71)
	if err != nil {
		t.Fatalf("GetProductVariants failed: %v", err)
	}

	if len(variants) != 2 {
		t.Errorf("Expected 2 variants, got %d", len(variants))
	}

	if variants[0].Size != "12x16" {
		t.Errorf("Expected variant size '12x16', got '%s'", variants[0].Size)
	}
}

// TestPrintfulClient_ConvertProduct tests product conversion
func TestPrintfulClient_ConvertProduct(t *testing.T) {
	mockProductSvc := &mockProductService{}
	client := NewPrintfulClient("test-api-key", mockProductSvc)

	printfulProduct := &models.PrintfulProduct{
		ID:          71,
		Name:        "Canvas Print",
		Description: "High-quality canvas",
		Category:    "Wall Art",
	}

	variants := []models.PrintfulVariant{
		{
			ID:        4012,
			ProductID: 71,
			Name:      "12x16 Canvas",
			Size:      "12x16",
			Color:     "White",
			Price:     "19.95",
			InStock:   true,
		},
	}

	productInput, err := client.convertPrintfulProduct(printfulProduct, variants)
	if err != nil {
		t.Fatalf("convertPrintfulProduct failed: %v", err)
	}

	if productInput.Title != "Canvas Print" {
		t.Errorf("Expected title 'Canvas Print', got '%s'", productInput.Title)
	}

	if len(productInput.Variants) != 1 {
		t.Errorf("Expected 1 variant, got %d", len(productInput.Variants))
	}

	if productInput.Variants[0].Price != 19.95 {
		t.Errorf("Expected price 19.95, got %.2f", productInput.Variants[0].Price)
	}

	if productInput.FulfillmentData.PartnerID != "printful" {
		t.Errorf("Expected partner ID 'printful', got '%s'", productInput.FulfillmentData.PartnerID)
	}

	// Check that tags include category and printful
	hasCategory := false
	hasPrintful := false
	for _, tag := range productInput.Tags {
		if tag == "Wall Art" {
			hasCategory = true
		}
		if tag == "printful" {
			hasPrintful = true
		}
	}
	if !hasCategory || !hasPrintful {
		t.Errorf("Expected tags to include 'Wall Art' and 'printful', got %v", productInput.Tags)
	}
}

// TestPrintfulClient_ConvertProductNoVariants tests error handling
func TestPrintfulClient_ConvertProductNoVariants(t *testing.T) {
	mockProductSvc := &mockProductService{}
	client := NewPrintfulClient("test-api-key", mockProductSvc)

	printfulProduct := &models.PrintfulProduct{
		ID:   71,
		Name: "Canvas Print",
	}

	_, err := client.convertPrintfulProduct(printfulProduct, []models.PrintfulVariant{})
	if err == nil {
		t.Error("Expected error for product with no variants")
	}
}

// TestPrintfulClient_ConvertProductOutOfStock tests out of stock filtering
func TestPrintfulClient_ConvertProductOutOfStock(t *testing.T) {
	mockProductSvc := &mockProductService{}
	client := NewPrintfulClient("test-api-key", mockProductSvc)

	printfulProduct := &models.PrintfulProduct{
		ID:   71,
		Name: "Canvas Print",
	}

	variants := []models.PrintfulVariant{
		{
			ID:        4012,
			ProductID: 71,
			Name:      "12x16 Canvas",
			Price:     "19.95",
			InStock:   false, // Out of stock
		},
	}

	_, err := client.convertPrintfulProduct(printfulProduct, variants)
	if err == nil {
		t.Error("Expected error when all variants are out of stock")
	}
}

// TestPrintfulClient_ImportProduct tests product import with markup
func TestPrintfulClient_ImportProduct(t *testing.T) {
	productCreated := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/products/71" {
			response := models.PrintfulAPIResponse{
				Code: 200,
				Result: map[string]interface{}{
					"product": models.PrintfulProduct{
						ID:          71,
						Name:        "Canvas Print",
						Description: "High-quality canvas",
					},
					"variants": []models.PrintfulVariant{
						{
							ID:        4012,
							ProductID: 71,
							Name:      "12x16 Canvas",
							Price:     "20.00",
							InStock:   true,
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	mockProductSvc := &mockProductService{
		createProductFunc: func(ctx context.Context, product *models.Product) error {
			productCreated = true
			// Verify markup was applied (200% markup on $20 = $60)
			expectedPrice := 60.0
			if product.Price != expectedPrice {
				t.Errorf("Expected price %.2f after 200%% markup, got %.2f", expectedPrice, product.Price)
			}
			return nil
		},
	}

	client := NewPrintfulClient("test-api-key", mockProductSvc)
	client.baseURL = server.URL

	req := &models.PrintfulProductImportRequest{
		PrintfulProductID: "71",
		MarkupPercentage:  200.0, // 200% markup
		Title:             "Custom Canvas Print",
	}

	product, err := client.ImportProduct(context.Background(), req)
	if err != nil {
		t.Fatalf("ImportProduct failed: %v", err)
	}

	if !productCreated {
		t.Error("Expected product to be created")
	}

	if product.Title != "Custom Canvas Print" {
		t.Errorf("Expected custom title, got '%s'", product.Title)
	}
}

// TestProductInputToProduct tests conversion helper
func TestProductInputToProduct(t *testing.T) {
	mockProductSvc := &mockProductService{}
	client := NewPrintfulClient("test-api-key", mockProductSvc)

	input := &models.ProductInput{
		Title:       "Test Product",
		Description: "Test Description",
		Price:       29.99,
		SKU:         "TEST-SKU",
		Status:      "active",
		Inventory:   100,
		Tags:        []string{"test", "printful"},
		Variants: []models.ProductVariantInput{
			{
				Title:     "Variant 1",
				Price:     29.99,
				SKU:       "VAR-1",
				Inventory: 50,
			},
		},
	}

	product := client.productInputToProduct(input)

	if product.Title != input.Title {
		t.Errorf("Expected title '%s', got '%s'", input.Title, product.Title)
	}

	if len(product.Variants) != 1 {
		t.Errorf("Expected 1 variant, got %d", len(product.Variants))
	}

	if product.Variants[0].Title != "Variant 1" {
		t.Errorf("Expected variant title 'Variant 1', got '%s'", product.Variants[0].Title)
	}
}
