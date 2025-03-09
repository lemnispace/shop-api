package routers

import (
	"log"
	"net/http"

	"github.com/lemnispace/shop-api/internal/handlers"
	"github.com/lemnispace/shop-api/internal/services"
)

// ServiceFactory is a function that creates product and collection services
type ServiceFactory func() (services.ProductService, services.CollectionService)

// defaultServiceFactory is a placeholder that logs an error if no factory is set
func defaultServiceFactory() (services.ProductService, services.CollectionService) {
	log.Fatalf("No service factory configured! You must call SetServiceFactory with a valid DynamoDB configuration before using the API.")
	// This line will never be reached due to log.Fatalf, but is needed for compilation
	return nil, nil
}

// Current service factory - must be replaced by the application
var currentServiceFactory ServiceFactory = defaultServiceFactory

// SetServiceFactory allows the application to set a custom service factory
func SetServiceFactory(factory ServiceFactory) {
	currentServiceFactory = factory
}

func InitRouter() *http.ServeMux {
	router := http.NewServeMux()

	// API versioning prefix
	apiPrefix := "/v1"

	// Initialize services
	initServices()

	// Product routes
	router.HandleFunc(apiPrefix+"/products", handlers.ProductsHandler)
	router.HandleFunc(apiPrefix+"/products/", handlers.ProductDetailHandler) // This now handles variants and images
	router.HandleFunc(apiPrefix+"/products/count", handlers.ProductCountHandler)
	router.HandleFunc(apiPrefix+"/products/variants", handlers.ProductVariantsHandler)

	// Collection routes
	router.HandleFunc(apiPrefix+"/collections", handlers.CollectionsHandler)
	router.HandleFunc(apiPrefix+"/collections/", handlers.CollectionDetailHandler)
	router.HandleFunc(apiPrefix+"/collections/count", handlers.CollectionCountHandler)

	// TODO: Add routes for other resources (Cart, Orders, etc.)

	return router
}

// initServices initializes all the services using the current factory
func initServices() {
	productService, collectionService := currentServiceFactory()

	// Register services with the handlers
	handlers.SetProductService(productService)
	handlers.SetCollectionService(collectionService)
}
