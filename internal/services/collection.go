package services

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lemnispace/shop-api/internal/models"
)

var (
	ErrCollectionNotFound = errors.New("collection not found")
)

// CollectionListResult represents the result of a collection list operation with pagination
type CollectionListResult struct {
	Collections []models.Collection
	NextCursor  string
}

// CollectionService defines the interface for collection operations
type CollectionService interface {
	GetCollection(ctx context.Context, id string) (*models.Collection, error)
	CreateCollection(ctx context.Context, collection *models.Collection) error
	UpdateCollection(ctx context.Context, collection *models.Collection) error
	DeleteCollection(ctx context.Context, id string) error
	ListCollections(ctx context.Context, limit int, cursor string, filters map[string]interface{}, sortKey, sortOrder string) (*CollectionListResult, error)
	CountCollections(ctx context.Context, filters map[string]interface{}) (int, error)
	AddProductToCollection(ctx context.Context, collectionID, productID string) error
	RemoveProductFromCollection(ctx context.Context, collectionID, productID string) error
	ListCollectionProducts(ctx context.Context, collectionID string, limit int, cursor string) ([]models.Product, string, error)
}

// InMemoryCollectionService is an in-memory implementation of the CollectionService
type InMemoryCollectionService struct {
	mu          sync.RWMutex
	collections map[string]*models.Collection
	lastID      int
	// We need access to products to populate collection with products
	productService ProductService
}

// NewInMemoryCollectionService creates a new in-memory collection service
func NewInMemoryCollectionService(productService ProductService) CollectionService {
	service := &InMemoryCollectionService{
		collections:    make(map[string]*models.Collection),
		lastID:         2000, // Start collection IDs at 2000
		productService: productService,
	}

	// Add some sample collections
	service.createSampleCollections()

	return service
}

// GetCollection retrieves a collection by ID
func (s *InMemoryCollectionService) GetCollection(ctx context.Context, id string) (*models.Collection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	collection, exists := s.collections[id]
	if !exists {
		return nil, ErrCollectionNotFound
	}

	// Return a copy to prevent concurrent modification
	collectionCopy := *collection

	// Load products for the collection
	if len(collection.Products) == 0 && len(collectionCopy.Products) == 0 {
		// We need to load products if they aren't already loaded
		collectionCopy.Products = make([]models.Product, 0)
	}

	return &collectionCopy, nil
}

// CreateCollection creates a new collection
func (s *InMemoryCollectionService) CreateCollection(ctx context.Context, collection *models.Collection) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Set timestamps
	now := time.Now()
	collection.CreatedAt = now
	collection.UpdatedAt = now

	// Generate ID if not provided
	if collection.ID == "" {
		s.lastID++
		collection.ID = strconv.Itoa(s.lastID)
	}

	// Store the collection (make a copy to prevent reference issues)
	collectionCopy := *collection

	// Initialize products slice if nil
	if collectionCopy.Products == nil {
		collectionCopy.Products = make([]models.Product, 0)
	}

	s.collections[collection.ID] = &collectionCopy

	// Update the original collection with the ID
	*collection = collectionCopy

	return nil
}

// UpdateCollection updates an existing collection
func (s *InMemoryCollectionService) UpdateCollection(ctx context.Context, collection *models.Collection) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if collection exists
	existingCollection, exists := s.collections[collection.ID]
	if !exists {
		return ErrCollectionNotFound
	}

	// Preserve creation timestamp
	collection.CreatedAt = existingCollection.CreatedAt
	collection.UpdatedAt = time.Now()

	// Store the updated collection (make a copy to prevent reference issues)
	collectionCopy := *collection

	// Initialize products slice if nil
	if collectionCopy.Products == nil {
		collectionCopy.Products = make([]models.Product, 0)
	}

	s.collections[collection.ID] = &collectionCopy

	// Update the original collection
	*collection = collectionCopy

	return nil
}

// DeleteCollection deletes a collection by ID
func (s *InMemoryCollectionService) DeleteCollection(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if collection exists
	if _, exists := s.collections[id]; !exists {
		return ErrCollectionNotFound
	}

	// Delete the collection
	delete(s.collections, id)

	return nil
}

// ListCollections lists collections with pagination, filtering, and sorting
func (s *InMemoryCollectionService) ListCollections(ctx context.Context, limit int, cursor string, filters map[string]interface{}, sortKey, sortOrder string) (*CollectionListResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get all collections
	allCollections := make([]models.Collection, 0, len(s.collections))
	for _, c := range s.collections {
		// Apply filters
		if matchesCollectionFilters(*c, filters) {
			allCollections = append(allCollections, *c)
		}
	}

	// Sort collections
	sortCollections(allCollections, sortKey, sortOrder)

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
	if endIndex > len(allCollections) {
		endIndex = len(allCollections)
	}

	var result []models.Collection
	if startIndex < len(allCollections) {
		result = allCollections[startIndex:endIndex]
	} else {
		result = []models.Collection{}
	}

	// Create cursor for next page
	var nextCursor string
	if endIndex < len(allCollections) {
		nextCursor = strconv.Itoa(endIndex)
	}

	return &CollectionListResult{
		Collections: result,
		NextCursor:  nextCursor,
	}, nil
}

// CountCollections returns the count of collections based on filters
func (s *InMemoryCollectionService) CountCollections(ctx context.Context, filters map[string]interface{}) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, c := range s.collections {
		if matchesCollectionFilters(*c, filters) {
			count++
		}
	}

	return count, nil
}

// AddProductToCollection adds a product to a collection
func (s *InMemoryCollectionService) AddProductToCollection(ctx context.Context, collectionID, productID string) error {
	// First verify the product exists
	product, err := s.productService.GetProduct(ctx, productID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if collection exists
	collection, exists := s.collections[collectionID]
	if !exists {
		return ErrCollectionNotFound
	}

	// Check if product is already in the collection
	for _, p := range collection.Products {
		if p.ID == productID {
			// Product already in collection, nothing to do
			return nil
		}
	}

	// Add the product to the collection
	collection.Products = append(collection.Products, *product)
	collection.UpdatedAt = time.Now()

	return nil
}

// RemoveProductFromCollection removes a product from a collection
func (s *InMemoryCollectionService) RemoveProductFromCollection(ctx context.Context, collectionID, productID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if collection exists
	collection, exists := s.collections[collectionID]
	if !exists {
		return ErrCollectionNotFound
	}

	// Find and remove the product
	for i, p := range collection.Products {
		if p.ID == productID {
			// Remove the product (preserve order)
			collection.Products = append(collection.Products[:i], collection.Products[i+1:]...)
			collection.UpdatedAt = time.Now()
			return nil
		}
	}

	// Product not found in collection, nothing to do
	return nil
}

// ListCollectionProducts lists the products in a collection with pagination
func (s *InMemoryCollectionService) ListCollectionProducts(ctx context.Context, collectionID string, limit int, cursor string) ([]models.Product, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if collection exists
	collection, exists := s.collections[collectionID]
	if !exists {
		return nil, "", ErrCollectionNotFound
	}

	// Get all products in the collection
	products := collection.Products

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
	if endIndex > len(products) {
		endIndex = len(products)
	}

	var result []models.Product
	if startIndex < len(products) {
		result = products[startIndex:endIndex]
	} else {
		result = []models.Product{}
	}

	// Create cursor for next page
	var nextCursor string
	if endIndex < len(products) {
		nextCursor = strconv.Itoa(endIndex)
	}

	return result, nextCursor, nil
}

// Helper methods

// matchesCollectionFilters checks if a collection matches the given filters
func matchesCollectionFilters(collection models.Collection, filters map[string]interface{}) bool {
	if len(filters) == 0 {
		return true
	}

	for key, value := range filters {
		switch key {
		case "title":
			if title, ok := value.(string); ok && !strings.Contains(strings.ToLower(collection.Title), strings.ToLower(title)) {
				return false
			}
		}
	}

	return true
}

// sortCollections sorts collections by the given key and order
func sortCollections(collections []models.Collection, sortKey, sortOrder string) {
	sort.Slice(collections, func(i, j int) bool {
		less := false

		switch sortKey {
		case "created_at":
			less = collections[i].CreatedAt.Before(collections[j].CreatedAt)
		case "updated_at":
			less = collections[i].UpdatedAt.Before(collections[j].UpdatedAt)
		case "title":
			less = collections[i].Title < collections[j].Title
		default:
			// Default sort by created_at
			less = collections[i].CreatedAt.Before(collections[j].CreatedAt)
		}

		// Reverse for descending order
		if sortOrder == "desc" {
			return !less
		}
		return less
	})
}

// createSampleCollections creates sample collections for testing
func (s *InMemoryCollectionService) createSampleCollections() {
	// Sample collection 1: Featured Products
	featuredCollection := &models.Collection{
		ID:          "2001",
		Title:       "Featured Products",
		Description: "Our hand-picked selection of featured products",
		Products:    []models.Product{},
		CreatedAt:   time.Now().Add(-48 * time.Hour),
		UpdatedAt:   time.Now().Add(-24 * time.Hour),
	}

	// Sample collection 2: New Arrivals
	newArrivalsCollection := &models.Collection{
		ID:          "2002",
		Title:       "New Arrivals",
		Description: "Check out our latest products",
		Products:    []models.Product{},
		CreatedAt:   time.Now().Add(-24 * time.Hour),
		UpdatedAt:   time.Now().Add(-12 * time.Hour),
	}

	// Sample collection 3: Clearance
	clearanceCollection := &models.Collection{
		ID:          "2003",
		Title:       "Clearance",
		Description: "Great deals on products that are being discontinued",
		Products:    []models.Product{},
		CreatedAt:   time.Now().Add(-72 * time.Hour),
		UpdatedAt:   time.Now().Add(-48 * time.Hour),
	}

	// Add collections to storage
	s.collections[featuredCollection.ID] = featuredCollection
	s.collections[newArrivalsCollection.ID] = newArrivalsCollection
	s.collections[clearanceCollection.ID] = clearanceCollection

	// Attempt to populate with sample products if we can
	ctx := context.Background()

	// Add the first available product to Featured Products collection
	allProductsResult, err := s.productService.ListProducts(ctx, 1, "", nil, "created_at", "desc")
	if err == nil && len(allProductsResult.Products) > 0 {
		product := allProductsResult.Products[0]
		featuredCollection.Products = append(featuredCollection.Products, product)
	}
}
