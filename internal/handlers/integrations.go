package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/internal/services"
)

// Package-level service variable
var printfulService services.PrintfulService

// SetPrintfulService sets the Printful service for handlers
func SetPrintfulService(service services.PrintfulService) {
	printfulService = service
}

// SyncPrintfulCatalog handles POST /v1/integrations/printful/sync
func SyncPrintfulCatalog(c *gin.Context) {
	ctx := c.Request.Context()

	// Start async sync job
	go func() {
		job, err := printfulService.SyncCatalog(ctx)
		if err != nil {
			log.Printf("[ERROR] Printful sync failed: %v", err)
			return
		}
		log.Printf("[INFO] Printful sync completed: job_id=%s, synced=%d/%d",
			job.ID, job.ItemsProcessed, job.ItemsTotal)
	}()

	// Return immediate response
	c.JSON(http.StatusAccepted, gin.H{
		"message": "Catalog sync started",
		"status":  "processing",
	})
}

// GetSyncStatus handles GET /v1/integrations/printful/sync/:id
func GetSyncStatus(c *gin.Context) {
	syncID := c.Param("id")

	// For now, return a simple response
	// In a production system, you'd track sync jobs in a database
	c.JSON(http.StatusOK, gin.H{
		"id":             syncID,
		"type":           "printful_products",
		"status":         "completed",
		"progress":       100,
		"itemsProcessed": 0,
		"itemsTotal":     0,
	})
}

// ListPrintfulProducts handles GET /v1/integrations/printful/products
func ListPrintfulProducts(c *gin.Context) {
	// Check if service is initialized
	if printfulService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"code":    "SERVICE_UNAVAILABLE",
				"message": "Printful integration is not configured. Please set PRINTFUL_API_KEY environment variable.",
			},
		})
		return
	}

	ctx := c.Request.Context()

	// Get query parameters
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	// Fetch products from Printful
	products, err := printfulService.GetProducts(ctx)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch Printful products: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "PRINTFUL_ERROR",
				"message": "Failed to fetch products from Printful",
			},
		})
		return
	}

	// Apply limit
	if len(products) > limit {
		products = products[:limit]
	}

	// Convert to response format
	responseProducts := make([]map[string]interface{}, 0, len(products))
	for _, p := range products {
		// Skip discontinued or non-syncable products
		if p.IsDiscontinued || !p.IsSyncable {
			continue
		}

		product := map[string]interface{}{
			"id":              strconv.Itoa(p.ID),
			"title":           p.Name,
			"description":     p.Description,
			"category":        p.Category,
			"mockupImageUrl":  p.Thumbnail,
			"variantCount":    p.Variants,
		}

		responseProducts = append(responseProducts, product)
	}

	c.JSON(http.StatusOK, gin.H{
		"products": responseProducts,
		"pagination": gin.H{
			"hasMore": false,
		},
	})
}

// GetPrintfulProduct handles GET /v1/integrations/printful/products/:id
func GetPrintfulProduct(c *gin.Context) {
	// Check if service is initialized
	if printfulService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"code":    "SERVICE_UNAVAILABLE",
				"message": "Printful integration is not configured. Please set PRINTFUL_API_KEY environment variable.",
			},
		})
		return
	}

	ctx := c.Request.Context()

	productIDStr := c.Param("id")
	productID, err := strconv.Atoi(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_PRODUCT_ID",
				"message": "Invalid product ID",
			},
		})
		return
	}

	// Fetch product from Printful
	product, err := printfulService.GetProduct(ctx, productID)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch Printful product %d: %v", productID, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "PRODUCT_NOT_FOUND",
				"message": "Product not found in Printful catalog",
			},
		})
		return
	}

	// Fetch variants
	variants, err := printfulService.GetProductVariants(ctx, productID)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch variants for product %d: %v", productID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "PRINTFUL_ERROR",
				"message": "Failed to fetch product variants",
			},
		})
		return
	}

	// Convert variants to response format
	responseVariants := make([]map[string]interface{}, 0, len(variants))
	for _, v := range variants {
		if !v.InStock {
			continue
		}

		variant := map[string]interface{}{
			"id":    strconv.Itoa(v.ID),
			"title": v.Name,
			"price": v.Price,
			"size":  v.Size,
			"color": v.Color,
			"image": v.Image,
		}
		responseVariants = append(responseVariants, variant)
	}

	c.JSON(http.StatusOK, gin.H{
		"id":              strconv.Itoa(product.ID),
		"title":           product.Name,
		"description":     product.Description,
		"category":        product.Category,
		"mockupImageUrl":  product.Thumbnail,
		"variants":        responseVariants,
	})
}

// ImportPrintfulProduct handles POST /v1/integrations/printful/products/import
func ImportPrintfulProduct(c *gin.Context) {
	ctx := c.Request.Context()

	var req models.PrintfulProductImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
				"details": err.Error(),
			},
		})
		return
	}

	// Validate required fields first (before checking service)
	if req.PrintfulProductID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "MISSING_FIELD",
				"message": "printfulProductId is required",
			},
		})
		return
	}

	if req.MarkupPercentage < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_MARKUP",
				"message": "markupPercentage must be non-negative",
			},
		})
		return
	}

	// Check if service is initialized after validation
	if printfulService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"code":    "SERVICE_UNAVAILABLE",
				"message": "Printful integration is not configured. Please set PRINTFUL_API_KEY environment variable.",
			},
		})
		return
	}

	// Import product
	product, err := printfulService.ImportProduct(ctx, &req)
	if err != nil {
		log.Printf("[ERROR] Failed to import Printful product %s: %v", req.PrintfulProductID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "IMPORT_FAILED",
				"message": "Failed to import product",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, product)
}

// SubmitPrintfulOrder handles POST /v1/integrations/printful/orders
func SubmitPrintfulOrder(c *gin.Context) {
	// Check if service is initialized
	if printfulService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"code":    "SERVICE_UNAVAILABLE",
				"message": "Printful integration is not configured. Please set PRINTFUL_API_KEY environment variable.",
			},
		})
		return
	}

	ctx := c.Request.Context()

	var req models.PrintfulOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
				"details": err.Error(),
			},
		})
		return
	}

	// Validate required fields
	if req.ExternalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "MISSING_FIELD",
				"message": "external_id is required",
			},
		})
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "NO_ITEMS",
				"message": "Order must contain at least one item",
			},
		})
		return
	}

	// Create order with Printful
	order, err := printfulService.CreateOrder(ctx, &req)
	if err != nil {
		log.Printf("[ERROR] Failed to create Printful order %s: %v", req.ExternalID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "ORDER_CREATION_FAILED",
				"message": "Failed to create order with Printful",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, order)
}

// GetPrintfulOrder handles GET /v1/integrations/printful/orders/:id
func GetPrintfulOrder(c *gin.Context) {
	// Check if service is initialized
	if printfulService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"code":    "SERVICE_UNAVAILABLE",
				"message": "Printful integration is not configured. Please set PRINTFUL_API_KEY environment variable.",
			},
		})
		return
	}

	ctx := c.Request.Context()
	orderID := c.Param("id")

	order, err := printfulService.GetOrder(ctx, orderID)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch Printful order %s: %v", orderID, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "ORDER_NOT_FOUND",
				"message": "Order not found in Printful",
			},
		})
		return
	}

	c.JSON(http.StatusOK, order)
}

// ConfirmPrintfulOrder handles POST /v1/integrations/printful/orders/:id/confirm
func ConfirmPrintfulOrder(c *gin.Context) {
	// Check if service is initialized
	if printfulService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"code":    "SERVICE_UNAVAILABLE",
				"message": "Printful integration is not configured. Please set PRINTFUL_API_KEY environment variable.",
			},
		})
		return
	}

	ctx := c.Request.Context()
	orderID := c.Param("id")

	order, err := printfulService.ConfirmOrder(ctx, orderID)
	if err != nil {
		log.Printf("[ERROR] Failed to confirm Printful order %s: %v", orderID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "CONFIRMATION_FAILED",
				"message": "Failed to confirm order with Printful",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, order)
}

// CancelPrintfulOrder handles DELETE /v1/integrations/printful/orders/:id
func CancelPrintfulOrder(c *gin.Context) {
	// Check if service is initialized
	if printfulService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"code":    "SERVICE_UNAVAILABLE",
				"message": "Printful integration is not configured. Please set PRINTFUL_API_KEY environment variable.",
			},
		})
		return
	}

	ctx := c.Request.Context()
	orderID := c.Param("id")

	order, err := printfulService.CancelOrder(ctx, orderID)
	if err != nil {
		log.Printf("[ERROR] Failed to cancel Printful order %s: %v", orderID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "CANCELLATION_FAILED",
				"message": "Failed to cancel order with Printful",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, order)
}
