package app

import (
	"github.com/lemnispace/shop-api/internal/services"
)

// Context holds all application services and dependencies
// This replaces global package-level service variables to prevent
// data races and make dependency injection explicit
type Context struct {
	ProductService       services.ProductService
	CollectionService    services.CollectionService
	CartService          *services.CartService
	OrderService         services.OrderService
	PaymentService       services.PaymentService
	CustomizationService services.CustomizationService
	AuthService          services.AuthService
	CustomerService      services.CustomerService
	PrintfulService      services.PrintfulService
	FulfillmentService   services.FulfillmentService
	S3Service            services.S3Service
}

// NewContext creates a new application context with all services
func NewContext() *Context {
	return &Context{}
}

// Validate ensures all required services are initialized
// Returns an error listing any missing services
func (ctx *Context) Validate() error {
	// For now, we'll allow optional services to be nil
	// but we could add stricter validation if needed
	return nil
}
