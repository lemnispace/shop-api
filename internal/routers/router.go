package routers

import (
	"log"
	"net/http"

	"github.com/lemnispace/shop-api/internal/handlers"
	"github.com/lemnispace/shop-api/internal/services"
)

// ServiceFactory is a function that creates product and collection services
type ServiceFactory func() (services.ProductService, services.CollectionService, *services.CartService)

// defaultServiceFactory is a placeholder that logs an error if no factory is set
func defaultServiceFactory() (services.ProductService, services.CollectionService, *services.CartService) {
	log.Fatalf("No service factory configured! You must call SetServiceFactory with a valid DynamoDB configuration before using the API.")
	// This line will never be reached due to log.Fatalf, but is needed for compilation
	return nil, nil, nil
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

	// Cart routes
	router.HandleFunc(apiPrefix+"/cart", handlers.CartHandler)        // POST: create cart, GET ?customer=xxx: list customer carts
	router.HandleFunc(apiPrefix+"/cart/", handlers.CartDetailHandler) // GET: get cart, POST /items: add item, etc.

	// TODO: Add routes for other resources (Orders, Fulfillments, etc.)

	return router
}

// initServices initializes all the services using the current factory
func initServices() {
	productService, collectionService, cartService := currentServiceFactory()

	// Register services with the handlers
	handlers.SetProductService(productService)
	handlers.SetCollectionService(collectionService)

	// Register cart service with the handlers
	if cartService != nil {
		handlers.SetCartService(cartService)
	}
}
