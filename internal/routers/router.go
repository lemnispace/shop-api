package routers

import (
	"net/http"

	"github.com/lemnispace/shop-api/internal/handlers"
)

func InitRouter() *http.ServeMux {
	router := http.NewServeMux()

	// API versioning prefix
	apiPrefix := "/v1"

	// Product routes
	router.HandleFunc(apiPrefix+"/products", handlers.ProductsHandler)
	router.HandleFunc(apiPrefix+"/products/", handlers.ProductDetailHandler)

	// TODO: Add routes for other resources

	return router
}
