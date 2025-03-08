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
	initProductService()

	// Product routes
	router.HandleFunc(apiPrefix+"/products", handlers.ProductsHandler)
	router.HandleFunc(apiPrefix+"/products/", handlers.ProductDetailHandler)
	router.HandleFunc(apiPrefix+"/products/count", handlers.ProductCountHandler)
	// Product variants endpoint
	router.HandleFunc(apiPrefix+"/products/variants", handlers.ProductVariantsHandler)

	// TODO: Add routes for other resources

	return router
}

// stubProductService is a simple in-memory implementation for development
// type stubProductService struct {
// 	products map[string]*models.Product
// }

// initProductService initializes the product service
func initProductService() {
	// Create and initialize the in-memory product service
	inMemoryService := services.NewInMemoryProductService()

	// Register the service with the handlers
	handlers.SetProductService(inMemoryService)
}
