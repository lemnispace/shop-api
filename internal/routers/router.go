package routers

import (
	"net/http"

	"github.com/lemnispace/shop-api/internal/handlers"
	"github.com/lemnispace/shop-api/internal/services"
	// "github.com/lemnispace/shop-api/internal/models"
)

func InitRouter() *http.ServeMux {
	router := http.NewServeMux()

	// API versioning prefix
	apiPrefix := "/v1"

	// Initialize in-memory product service for development
	initServices()

	// Product routes
	router.HandleFunc(apiPrefix+"/products", handlers.ProductsHandler)
	router.HandleFunc(apiPrefix+"/products/", handlers.ProductDetailHandler)
	router.HandleFunc(apiPrefix+"/products/count", handlers.ProductCountHandler)
	// Product variants endpoint
	router.HandleFunc(apiPrefix+"/products/variants", handlers.ProductVariantsHandler)

	// Collection routes
	router.HandleFunc(apiPrefix+"/collections", handlers.CollectionsHandler)
	router.HandleFunc(apiPrefix+"/collections/", handlers.CollectionDetailHandler)
	router.HandleFunc(apiPrefix+"/collections/count", handlers.CollectionCountHandler)

	// TODO: Add routes for other resources

	return router
}

// stubProductService is a simple in-memory implementation for development
// type stubProductService struct {
// 	products map[string]*models.Product
// }

// initServices initializes all the services
func initServices() {
	// Create and initialize the in-memory product service
	productService := services.NewInMemoryProductService()

	// Register the product service with the handlers
	handlers.SetProductService(productService)

	// Create and initialize the in-memory collection service
	// Note that the collection service depends on the product service
	collectionService := services.NewInMemoryCollectionService(productService)

	// Register the collection service with the handlers
	handlers.SetCollectionService(collectionService)
}
