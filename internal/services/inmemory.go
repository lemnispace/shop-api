package services

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/lemnispace/shop-api/internal/models"
)

// InMemoryProductService is an in-memory implementation of the ProductService
type InMemoryProductService struct {
	mu       sync.RWMutex
	products map[string]*models.Product
	lastID   int
}

// NewInMemoryProductService creates a new in-memory product service
func NewInMemoryProductService() ProductService {
	service := &InMemoryProductService{
		products: make(map[string]*models.Product),
		lastID:   1000, // Start with ID 1000
	}

	// Add some sample products
	service.createSampleProducts()

	return service
}

// GetProduct retrieves a product by ID
func (s *InMemoryProductService) GetProduct(ctx context.Context, id string) (*models.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	product, exists := s.products[id]
	if !exists {
		return nil, ErrProductNotFound
	}

	// Return a copy of the product to prevent concurrent modification
	productCopy := *product
	return &productCopy, nil
}

// CreateProduct creates a new product
func (s *InMemoryProductService) CreateProduct(ctx context.Context, product *models.Product) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Set creation timestamp
	now := time.Now()
	product.CreatedAt = now
	product.UpdatedAt = now

	// Generate ID if not provided
	if product.ID == "" {
		s.lastID++
		product.ID = strconv.Itoa(s.lastID)
	}

	// Set product information in variants
	for i := range product.Variants {
		product.Variants[i].ProductID = product.ID
		product.Variants[i].ProductTitle = product.Title

		// Generate variant ID if not provided
		if product.Variants[i].ID == "" {
			s.lastID++
			product.Variants[i].ID = strconv.Itoa(s.lastID)
		}
	}

	// Store the product (make a copy to prevent reference issues)
	productCopy := *product
	s.products[product.ID] = &productCopy

	// Update the original product with the ID
	*product = productCopy

	return nil
}

// UpdateProduct updates an existing product
func (s *InMemoryProductService) UpdateProduct(ctx context.Context, product *models.Product) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if product exists
	existingProduct, exists := s.products[product.ID]
	if !exists {
		return ErrProductNotFound
	}

	// Preserve creation timestamp
	product.CreatedAt = existingProduct.CreatedAt
	product.UpdatedAt = time.Now()

	// Set product information in variants
	for i := range product.Variants {
		product.Variants[i].ProductID = product.ID
		product.Variants[i].ProductTitle = product.Title

		// Generate variant ID if not provided
		if product.Variants[i].ID == "" {
			s.lastID++
			product.Variants[i].ID = strconv.Itoa(s.lastID)
		}
	}

	// Store the updated product (make a copy to prevent reference issues)
	productCopy := *product
	s.products[product.ID] = &productCopy

	// Update the original product
	*product = productCopy

	return nil
}

// DeleteProduct deletes a product by ID
func (s *InMemoryProductService) DeleteProduct(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if product exists
	if _, exists := s.products[id]; !exists {
		return ErrProductNotFound
	}

	// Delete the product
	delete(s.products, id)

	return nil
}

// ListProducts lists products with pagination, filtering, and sorting
func (s *InMemoryProductService) ListProducts(ctx context.Context, limit int, cursor string, filters map[string]interface{}, sortKey, sortOrder string) (*ProductListResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get all products
	allProducts := make([]models.Product, 0, len(s.products))
	for _, p := range s.products {
		// Apply filters
		if matchesFilters(*p, filters) {
			allProducts = append(allProducts, *p)
		}
	}

	// Sort products
	sortProducts(allProducts, sortKey, sortOrder)

	// Apply pagination
	startIndex := 0
	if cursor != "" {
		var err error
		startIndex, err = strconv.Atoi(cursor)
		if err != nil {
			startIndex = 0
		}
	}

	endIndex := startIndex + limit
	if endIndex > len(allProducts) {
		endIndex = len(allProducts)
	}

	var result []models.Product
	if startIndex < len(allProducts) {
		result = allProducts[startIndex:endIndex]
	} else {
		result = []models.Product{}
	}

	// Create cursor for next page
	var nextCursor string
	if endIndex < len(allProducts) {
		nextCursor = strconv.Itoa(endIndex)
	}

	return &ProductListResult{
		Products:   result,
		NextCursor: nextCursor,
	}, nil
}

// CountProducts returns the count of products based on filters
func (s *InMemoryProductService) CountProducts(ctx context.Context, filters map[string]interface{}) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, p := range s.products {
		if matchesFilters(*p, filters) {
			count++
		}
	}

	return count, nil
}

// ListProductVariants lists variants for a specific product with pagination
func (s *InMemoryProductService) ListProductVariants(ctx context.Context, productID string, limit int, cursor string) ([]models.ProductVariant, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get the product
	product, exists := s.products[productID]
	if !exists {
		return nil, "", ErrProductNotFound
	}

	// Get all variants
	variants := product.Variants

	// Apply pagination
	startIndex := 0
	if cursor != "" {
		var err error
		startIndex, err = strconv.Atoi(cursor)
		if err != nil {
			startIndex = 0
		}
	}

	endIndex := startIndex + limit
	if endIndex > len(variants) {
		endIndex = len(variants)
	}

	var result []models.ProductVariant
	if startIndex < len(variants) {
		result = variants[startIndex:endIndex]
	} else {
		result = []models.ProductVariant{}
	}

	// Create cursor for next page
	var nextCursor string
	if endIndex < len(variants) {
		nextCursor = strconv.Itoa(endIndex)
	}

	return result, nextCursor, nil
}

// ListAllVariants lists all variants across products with pagination and filtering
func (s *InMemoryProductService) ListAllVariants(ctx context.Context, limit int, cursor string, filters map[string]interface{}, sortKey, sortOrder string) ([]models.ProductVariant, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get all variants from all products
	allVariants := []models.ProductVariant{}
	for _, p := range s.products {
		// Apply product filters
		if matchesFilters(*p, filters) {
			for _, v := range p.Variants {
				variant := v
				variant.ProductID = p.ID
				variant.ProductTitle = p.Title
				allVariants = append(allVariants, variant)
			}
		}
	}

	// Apply pagination
	startIndex := 0
	if cursor != "" {
		var err error
		startIndex, err = strconv.Atoi(cursor)
		if err != nil {
			startIndex = 0
		}
	}

	endIndex := startIndex + limit
	if endIndex > len(allVariants) {
		endIndex = len(allVariants)
	}

	var result []models.ProductVariant
	if startIndex < len(allVariants) {
		result = allVariants[startIndex:endIndex]
	} else {
		result = []models.ProductVariant{}
	}

	// Create cursor for next page
	var nextCursor string
	if endIndex < len(allVariants) {
		nextCursor = strconv.Itoa(endIndex)
	}

	return result, nextCursor, nil
}

// createSampleProducts creates sample products for testing
func (s *InMemoryProductService) createSampleProducts() {
	// Sample product 1
	product1 := &models.Product{
		ID:          "1001",
		Title:       "Premium T-Shirt",
		Description: "A high-quality cotton t-shirt that's both comfortable and stylish",
		Price:       29.99,
		SKU:         "TS-PREMIUM-001",
		Status:      "active",
		Inventory:   100,
		Tags:        []string{"t-shirt", "clothing", "cotton", "premium"},
		CustomFields: map[string]interface{}{
			"material": "100% Cotton",
			"care":     "Machine wash cold",
		},
		Images: []models.Image{
			{
				ID:        "img1001",
				URL:       "https://example.com/images/premium-tshirt-front.jpg",
				Width:     800,
				Height:    600,
				AltText:   "Premium T-Shirt Front View",
				IsDefault: true,
				Position:  1,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:        "img1002",
				URL:       "https://example.com/images/premium-tshirt-back.jpg",
				Width:     800,
				Height:    600,
				AltText:   "Premium T-Shirt Back View",
				IsDefault: false,
				Position:  2,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
		Variants: []models.ProductVariant{
			{
				ID:           "2001",
				ProductID:    "1001",
				ProductTitle: "Premium T-Shirt",
				SKU:          "TS-PREMIUM-001-S-BLK",
				Title:        "Small / Black",
				Price:        29.99,
				Inventory:    20,
				Options: []models.VariantOption{
					{Name: "Size", Value: "Small"},
					{Name: "Color", Value: "Black"},
				},
				Dimensions: models.Dimensions{
					Height: 2.0,
					Width:  30.0,
					Length: 30.0,
					Weight: 0.2,
				},
				FulfillmentData: models.FulfillmentData{
					HSCode:           "6109.10.00",
					CountryOfOrigin:  "US",
					Harmonized:       true,
					RequiresShipping: true,
				},
			},
			{
				ID:           "2002",
				ProductID:    "1001",
				ProductTitle: "Premium T-Shirt",
				SKU:          "TS-PREMIUM-001-M-BLK",
				Title:        "Medium / Black",
				Price:        29.99,
				Inventory:    30,
				Options: []models.VariantOption{
					{Name: "Size", Value: "Medium"},
					{Name: "Color", Value: "Black"},
				},
				Dimensions: models.Dimensions{
					Height: 2.0,
					Width:  32.0,
					Length: 32.0,
					Weight: 0.22,
				},
				FulfillmentData: models.FulfillmentData{
					HSCode:           "6109.10.00",
					CountryOfOrigin:  "US",
					Harmonized:       true,
					RequiresShipping: true,
				},
			},
		},
		Dimensions: models.Dimensions{
			Height: 2.0,
			Width:  30.0,
			Length: 30.0,
			Weight: 0.2,
		},
		FulfillmentData: models.FulfillmentData{
			HSCode:           "6109.10.00",
			CountryOfOrigin:  "US",
			Harmonized:       true,
			RequiresShipping: true,
		},
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now().Add(-12 * time.Hour),
	}

	// Sample product 2
	product2 := &models.Product{
		ID:          "1002",
		Title:       "Designer Coffee Mug",
		Description: "An elegant ceramic coffee mug perfect for your morning brew",
		Price:       14.99,
		SKU:         "MUG-DESIGNER-001",
		Status:      "active",
		Inventory:   75,
		Tags:        []string{"mug", "coffee", "ceramic", "kitchen"},
		CustomFields: map[string]interface{}{
			"material":   "Ceramic",
			"capacity":   "12 oz",
			"dishwasher": true,
			"microwave":  true,
		},
		Images: []models.Image{
			{
				ID:        "img2001",
				URL:       "https://example.com/images/designer-mug.jpg",
				Width:     800,
				Height:    600,
				AltText:   "Designer Coffee Mug",
				IsDefault: true,
				Position:  1,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
		Variants: []models.ProductVariant{
			{
				ID:           "3001",
				ProductID:    "1002",
				ProductTitle: "Designer Coffee Mug",
				SKU:          "MUG-DESIGNER-001-BLK",
				Title:        "Black",
				Price:        14.99,
				Inventory:    25,
				Options: []models.VariantOption{
					{Name: "Color", Value: "Black"},
				},
				Dimensions: models.Dimensions{
					Height: 10.0,
					Width:  8.0,
					Length: 8.0,
					Weight: 0.35,
				},
				FulfillmentData: models.FulfillmentData{
					HSCode:           "6912.00.10",
					CountryOfOrigin:  "CN",
					Harmonized:       true,
					RequiresShipping: true,
				},
			},
			{
				ID:           "3002",
				ProductID:    "1002",
				ProductTitle: "Designer Coffee Mug",
				SKU:          "MUG-DESIGNER-001-WHT",
				Title:        "White",
				Price:        14.99,
				Inventory:    25,
				Options: []models.VariantOption{
					{Name: "Color", Value: "White"},
				},
				Dimensions: models.Dimensions{
					Height: 10.0,
					Width:  8.0,
					Length: 8.0,
					Weight: 0.35,
				},
				FulfillmentData: models.FulfillmentData{
					HSCode:           "6912.00.10",
					CountryOfOrigin:  "CN",
					Harmonized:       true,
					RequiresShipping: true,
				},
			},
			{
				ID:           "3003",
				ProductID:    "1002",
				ProductTitle: "Designer Coffee Mug",
				SKU:          "MUG-DESIGNER-001-RED",
				Title:        "Red",
				Price:        14.99,
				Inventory:    25,
				Options: []models.VariantOption{
					{Name: "Color", Value: "Red"},
				},
				Dimensions: models.Dimensions{
					Height: 10.0,
					Width:  8.0,
					Length: 8.0,
					Weight: 0.35,
				},
				FulfillmentData: models.FulfillmentData{
					HSCode:           "6912.00.10",
					CountryOfOrigin:  "CN",
					Harmonized:       true,
					RequiresShipping: true,
				},
			},
		},
		Dimensions: models.Dimensions{
			Height: 10.0,
			Width:  8.0,
			Length: 8.0,
			Weight: 0.35,
		},
		FulfillmentData: models.FulfillmentData{
			HSCode:           "6912.00.10",
			CountryOfOrigin:  "CN",
			Harmonized:       true,
			RequiresShipping: true,
		},
		CreatedAt: time.Now().Add(-48 * time.Hour),
		UpdatedAt: time.Now().Add(-24 * time.Hour),
	}

	// Sample product 3 (draft status for testing filters)
	product3 := &models.Product{
		ID:          "1003",
		Title:       "Upcoming Collection Hoodie",
		Description: "A premium hoodie that's coming soon",
		Price:       49.99,
		SKU:         "HD-UPCOMING-001",
		Status:      "draft",
		Inventory:   0,
		Tags:        []string{"hoodie", "clothing", "upcoming", "premium"},
		CustomFields: map[string]interface{}{
			"material": "80% Cotton, 20% Polyester",
			"care":     "Machine wash cold",
			"release":  "2023-12-15",
		},
		Dimensions: models.Dimensions{
			Height: 5.0,
			Width:  40.0,
			Length: 35.0,
			Weight: 0.6,
		},
		FulfillmentData: models.FulfillmentData{
			HSCode:           "6110.20.20",
			CountryOfOrigin:  "US",
			Harmonized:       true,
			RequiresShipping: true,
		},
		CreatedAt: time.Now().Add(-12 * time.Hour),
		UpdatedAt: time.Now().Add(-6 * time.Hour),
	}

	// Add products to storage
	s.products[product1.ID] = product1
	s.products[product2.ID] = product2
	s.products[product3.ID] = product3
}
