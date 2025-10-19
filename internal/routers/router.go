package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/lemnispace/shop-api/internal/handlers"
	"github.com/lemnispace/shop-api/internal/middleware"
	"github.com/lemnispace/shop-api/internal/services"
)

func InitRouter(authService services.AuthService) *gin.Engine {
	// Create a default gin router with Logger and Recovery middleware
	router := gin.Default()

	// Health check endpoint (no versioning)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "shop-api",
		})
	})

	// API versioning prefix
	apiPrefix := "/v1"

	// API routes group
	v1 := router.Group(apiPrefix)
	{
		// Product routes
		products := v1.Group("/products")
		{
			// Public read endpoints
			products.GET("", handlers.ListAllProducts)                     // GET /v1/products
			products.GET("/count", handlers.ProductCount)                  // GET /v1/products/count
			products.GET("/variants", handlers.ListAllVariants)            // GET /v1/products/variants
			products.GET("/:productId", handlers.GetProduct)               // GET /v1/products/:productId
			products.GET("/:productId/variants", handlers.ListProductVariants) // GET /v1/products/:productId/variants

			// Protected admin endpoints - require authentication
			// TODO: Add admin role check once Customer model supports roles
			protected := products.Group("")
			protected.Use(middleware.AuthMiddleware(authService))
			{
				protected.POST("", handlers.CreateProduct)                                              // POST /v1/products
				protected.PUT("/:productId", handlers.UpdateProduct)                                    // PUT /v1/products/:productId
				protected.DELETE("/:productId", handlers.DeleteProduct)                                 // DELETE /v1/products/:productId
				protected.POST("/:productId/variants", handlers.CreateProductVariant)                   // POST /v1/products/:productId/variants
				protected.PUT("/:productId/variants/:variantId", handlers.UpdateProductVariant)         // PUT /v1/products/:productId/variants/:variantId
				protected.DELETE("/:productId/variants/:variantId", handlers.DeleteProductVariant)      // DELETE /v1/products/:productId/variants/:variantId
				protected.POST("/:productId/variants/:variantId/images", handlers.AssociateImageWithVariant) // POST /v1/products/:productId/variants/:variantId/images
				protected.POST("/:productId/images", handlers.UploadProductImage)                       // POST /v1/products/:productId/images
			}
		}

		// Collection routes
		collections := v1.Group("/collections")
		{
			// Public read endpoints
			collections.GET("", handlers.ListAllCollections)                            // GET /v1/collections
			collections.GET("/count", handlers.CollectionCount)                         // GET /v1/collections/count
			collections.GET("/:collectionId", handlers.GetCollection)                   // GET /v1/collections/:collectionId
			collections.GET("/:collectionId/products", handlers.ListCollectionProducts) // GET /v1/collections/:collectionId/products

			// Protected admin endpoints - require authentication
			// TODO: Add admin role check once Customer model supports roles
			protectedCollections := collections.Group("")
			protectedCollections.Use(middleware.AuthMiddleware(authService))
			{
				protectedCollections.POST("", handlers.CreateCollection)                                       // POST /v1/collections
				protectedCollections.PUT("/:collectionId", handlers.UpdateCollection)                          // PUT /v1/collections/:collectionId
				protectedCollections.DELETE("/:collectionId", handlers.DeleteCollection)                       // DELETE /v1/collections/:collectionId
				protectedCollections.POST("/:collectionId/products", handlers.AddProductToCollection)          // POST /v1/collections/:collectionId/products
				protectedCollections.DELETE("/:collectionId/products", handlers.RemoveProductFromCollection)   // DELETE /v1/collections/:collectionId/products
			}
		}

		// Cart routes
		cart := v1.Group("/cart")
		{
			cart.POST("", handlers.CreateCart)                             // POST /v1/cart
			cart.GET("", handlers.GetCustomerCarts)                        // GET /v1/cart?customer=xxx
			cart.GET("/:cartId", handlers.GetCart)                         // GET /v1/cart/:cartId
			cart.POST("/:cartId/items", handlers.AddCartItem)              // POST /v1/cart/:cartId/items
			cart.PUT("/:cartId/items/:itemId", handlers.UpdateCartItem)    // PUT /v1/cart/:cartId/items/:itemId
			cart.DELETE("/:cartId/items/:itemId", handlers.RemoveCartItem) // DELETE /v1/cart/:cartId/items/:itemId
			cart.POST("/:cartId/checkout", handlers.GetCartCheckout)       // POST /v1/cart/:cartId/checkout
		}

		// Customization routes (protected - require authentication)
		customizations := v1.Group("/customizations")
		customizations.Use(middleware.AuthMiddleware(authService))
		{
			customizations.GET("/images", handlers.ListCustomizationImages)                     // GET /v1/customizations/images
			customizations.POST("/images", handlers.UploadCustomizationImage)                   // POST /v1/customizations/images
			customizations.GET("/images/:imageId", handlers.GetCustomizationImage)              // GET /v1/customizations/images/:imageId
			customizations.DELETE("/images/:imageId", handlers.DeleteCustomizationImage)        // DELETE /v1/customizations/images/:imageId
			customizations.POST("/images/:imageId/process", handlers.ProcessCustomizationImage) // POST /v1/customizations/images/:imageId/process
			customizations.POST("/images/:imageId/link", handlers.LinkImageToCartItem)          // POST /v1/customizations/images/:imageId/link
		}

		// Order routes - all require authentication
		orders := v1.Group("/orders")
		orders.Use(middleware.AuthMiddleware(authService))
		{
			orders.POST("", handlers.CreateOrder)                                 // POST /v1/orders
			orders.GET("", handlers.ListOrders)                                   // GET /v1/orders (supports ?customerId=xxx)
			orders.GET("/:orderId", handlers.GetOrder)                            // GET /v1/orders/:orderId
			orders.POST("/:orderId/cancel", handlers.CancelOrder)                 // POST /v1/orders/:orderId/cancel
			orders.POST("/:orderId/payment-intent", handlers.CreatePaymentIntent) // POST /v1/orders/:orderId/payment-intent
			orders.POST("/:orderId/confirm-payment", handlers.ConfirmPayment)     // POST /v1/orders/:orderId/confirm-payment

			// Admin-only endpoint - TODO: Add admin role check
			orders.PATCH("/:orderId", handlers.UpdateOrderStatus) // PATCH /v1/orders/:orderId
		}

		// Webhook routes
		webhooks := v1.Group("/webhooks")
		{
			webhooks.POST("/stripe", handlers.HandleStripeWebhook) // POST /v1/webhooks/stripe
		}

		// Integration routes - admin only
		integrations := v1.Group("/integrations")
		integrations.Use(middleware.AuthMiddleware(authService)) // TODO: Add admin role check
		{
			// Printful integration
			printful := integrations.Group("/printful")
			{
				// Catalog sync
				printful.POST("/sync", handlers.SyncPrintfulCatalog)             // POST /v1/integrations/printful/sync
				printful.GET("/sync/:id", handlers.GetSyncStatus)                // GET /v1/integrations/printful/sync/:id

				// Products
				printful.GET("/products", handlers.ListPrintfulProducts)          // GET /v1/integrations/printful/products
				printful.GET("/products/:id", handlers.GetPrintfulProduct)        // GET /v1/integrations/printful/products/:id
				printful.POST("/products/import", handlers.ImportPrintfulProduct) // POST /v1/integrations/printful/products/import

				// Orders
				printful.POST("/orders", handlers.SubmitPrintfulOrder)               // POST /v1/integrations/printful/orders
				printful.GET("/orders/:id", handlers.GetPrintfulOrder)               // GET /v1/integrations/printful/orders/:id
				printful.POST("/orders/:id/confirm", handlers.ConfirmPrintfulOrder)  // POST /v1/integrations/printful/orders/:id/confirm
				printful.DELETE("/orders/:id", handlers.CancelPrintfulOrder)         // DELETE /v1/integrations/printful/orders/:id
			}
		}

		// Customer routes
		customers := v1.Group("/customers")
		{
			// Public routes
			customers.POST("/register", handlers.RegisterCustomer) // POST /v1/customers/register
			customers.POST("/login", handlers.LoginCustomer)       // POST /v1/customers/login
			customers.POST("/refresh", handlers.RefreshToken)      // POST /v1/customers/refresh

			// Protected routes (require authentication)
			protected := customers.Group("")
			protected.Use(middleware.AuthMiddleware(authService))
			{
				protected.GET("/me", handlers.GetCustomerProfile)       // GET /v1/customers/me
				protected.PUT("/me", handlers.UpdateCustomerProfile)    // PUT /v1/customers/me
				protected.DELETE("/me", handlers.DeleteCustomerAccount) // DELETE /v1/customers/me
			}
		}
	}

	return router
}
