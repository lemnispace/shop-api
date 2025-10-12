package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lemnispace/shop-api/internal/models"
)

// PrintfulService defines the interface for interacting with the Printful API
type PrintfulService interface {
	// Catalog operations
	GetProducts(ctx context.Context) ([]models.PrintfulProduct, error)
	GetProduct(ctx context.Context, productID int) (*models.PrintfulProduct, error)
	GetVariant(ctx context.Context, variantID int) (*models.PrintfulVariant, error)
	GetProductVariants(ctx context.Context, productID int) ([]models.PrintfulVariant, error)

	// Order operations
	CreateOrder(ctx context.Context, order *models.PrintfulOrderRequest) (*models.PrintfulOrder, error)
	GetOrder(ctx context.Context, orderID string) (*models.PrintfulOrder, error)
	ConfirmOrder(ctx context.Context, orderID string) (*models.PrintfulOrder, error)
	CancelOrder(ctx context.Context, orderID string) (*models.PrintfulOrder, error)

	// Sync operations
	SyncCatalog(ctx context.Context) (*models.PrintfulSyncJob, error)
	ImportProduct(ctx context.Context, req *models.PrintfulProductImportRequest) (*models.Product, error)
}

// PrintfulClient implements the PrintfulService interface
type PrintfulClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	productSvc ProductService
}

// NewPrintfulClient creates a new Printful API client
func NewPrintfulClient(apiKey string, productService ProductService) *PrintfulClient {
	return &PrintfulClient{
		apiKey:  apiKey,
		baseURL: "https://api.printful.com",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		productSvc: productService,
	}
}

// makeRequest makes an HTTP request to the Printful API
func (c *PrintfulClient) makeRequest(ctx context.Context, method, endpoint string, body interface{}) (*models.PrintfulAPIResponse, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := c.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	log.Printf("[DEBUG] Printful API Request: %s %s", method, url)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Printf("[DEBUG] Printful API Response: %d - %s", resp.StatusCode, string(respBody))

	var apiResp models.PrintfulAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if apiResp.Code < 200 || apiResp.Code >= 300 {
		if apiResp.Error != nil {
			return nil, fmt.Errorf("printful API error: %s - %s", apiResp.Error.Reason, apiResp.Error.Message)
		}
		return nil, fmt.Errorf("printful API error: status code %d", apiResp.Code)
	}

	return &apiResp, nil
}

// GetProducts fetches all products from the Printful catalog
func (c *PrintfulClient) GetProducts(ctx context.Context) ([]models.PrintfulProduct, error) {
	resp, err := c.makeRequest(ctx, "GET", "/products", nil)
	if err != nil {
		return nil, err
	}

	var products []models.PrintfulProduct
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := json.Unmarshal(resultBytes, &products); err != nil {
		return nil, fmt.Errorf("failed to unmarshal products: %w", err)
	}

	return products, nil
}

// GetProduct fetches a single product by ID
func (c *PrintfulClient) GetProduct(ctx context.Context, productID int) (*models.PrintfulProduct, error) {
	endpoint := fmt.Sprintf("/products/%d", productID)
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var productData map[string]interface{}
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := json.Unmarshal(resultBytes, &productData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal product data: %w", err)
	}

	// The product is nested under "product" key
	productBytes, err := json.Marshal(productData["product"])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal product: %w", err)
	}

	var product models.PrintfulProduct
	if err := json.Unmarshal(productBytes, &product); err != nil {
		return nil, fmt.Errorf("failed to unmarshal product: %w", err)
	}

	return &product, nil
}

// GetVariant fetches a single variant by ID
func (c *PrintfulClient) GetVariant(ctx context.Context, variantID int) (*models.PrintfulVariant, error) {
	endpoint := fmt.Sprintf("/products/variant/%d", variantID)
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var variantData map[string]interface{}
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := json.Unmarshal(resultBytes, &variantData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal variant data: %w", err)
	}

	// The variant is nested under "variant" key
	variantBytes, err := json.Marshal(variantData["variant"])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal variant: %w", err)
	}

	var variant models.PrintfulVariant
	if err := json.Unmarshal(variantBytes, &variant); err != nil {
		return nil, fmt.Errorf("failed to unmarshal variant: %w", err)
	}

	return &variant, nil
}

// GetProductVariants fetches all variants for a product
func (c *PrintfulClient) GetProductVariants(ctx context.Context, productID int) ([]models.PrintfulVariant, error) {
	endpoint := fmt.Sprintf("/products/%d", productID)
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var productData map[string]interface{}
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := json.Unmarshal(resultBytes, &productData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal product data: %w", err)
	}

	// The variants are nested under "variants" key
	variantsBytes, err := json.Marshal(productData["variants"])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal variants: %w", err)
	}

	var variants []models.PrintfulVariant
	if err := json.Unmarshal(variantsBytes, &variants); err != nil {
		return nil, fmt.Errorf("failed to unmarshal variants: %w", err)
	}

	return variants, nil
}

// CreateOrder creates a new order with Printful
func (c *PrintfulClient) CreateOrder(ctx context.Context, order *models.PrintfulOrderRequest) (*models.PrintfulOrder, error) {
	resp, err := c.makeRequest(ctx, "POST", "/orders", order)
	if err != nil {
		return nil, err
	}

	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var printfulOrder models.PrintfulOrder
	if err := json.Unmarshal(resultBytes, &printfulOrder); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order: %w", err)
	}

	return &printfulOrder, nil
}

// GetOrder retrieves an order by ID (external ID or Printful ID)
func (c *PrintfulClient) GetOrder(ctx context.Context, orderID string) (*models.PrintfulOrder, error) {
	endpoint := fmt.Sprintf("/orders/%s", orderID)
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var printfulOrder models.PrintfulOrder
	if err := json.Unmarshal(resultBytes, &printfulOrder); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order: %w", err)
	}

	return &printfulOrder, nil
}

// ConfirmOrder confirms a draft order
func (c *PrintfulClient) ConfirmOrder(ctx context.Context, orderID string) (*models.PrintfulOrder, error) {
	endpoint := fmt.Sprintf("/orders/%s/confirm", orderID)
	resp, err := c.makeRequest(ctx, "POST", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var printfulOrder models.PrintfulOrder
	if err := json.Unmarshal(resultBytes, &printfulOrder); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order: %w", err)
	}

	return &printfulOrder, nil
}

// CancelOrder cancels an order
func (c *PrintfulClient) CancelOrder(ctx context.Context, orderID string) (*models.PrintfulOrder, error) {
	endpoint := fmt.Sprintf("/orders/%s", orderID)
	resp, err := c.makeRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var printfulOrder models.PrintfulOrder
	if err := json.Unmarshal(resultBytes, &printfulOrder); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order: %w", err)
	}

	return &printfulOrder, nil
}

// SyncCatalog synchronizes the Printful catalog with the shop-api database
func (c *PrintfulClient) SyncCatalog(ctx context.Context) (*models.PrintfulSyncJob, error) {
	job := &models.PrintfulSyncJob{
		ID:        fmt.Sprintf("sync_%d", time.Now().Unix()),
		Type:      "printful_products",
		Status:    "running",
		Progress:  0,
		StartedAt: time.Now(),
	}

	// Fetch all products from Printful
	products, err := c.GetProducts(ctx)
	if err != nil {
		job.Status = "failed"
		job.Error = err.Error()
		return job, err
	}

	job.ItemsTotal = len(products)
	successCount := 0

	// Import each product
	for i, printfulProduct := range products {
		// Skip discontinued products
		if printfulProduct.IsDiscontinued || !printfulProduct.IsSyncable {
			continue
		}

		// Fetch full product details with variants
		variants, err := c.GetProductVariants(ctx, printfulProduct.ID)
		if err != nil {
			log.Printf("[ERROR] Failed to get variants for product %d: %v", printfulProduct.ID, err)
			continue
		}

		// Convert Printful product to shop-api product
		productInput, err := c.convertPrintfulProduct(&printfulProduct, variants)
		if err != nil {
			log.Printf("[ERROR] Failed to convert product %d: %v", printfulProduct.ID, err)
			continue
		}

		// Convert ProductInput to Product for creation
		product := c.productInputToProduct(productInput)

		// Create or update product in database
		err = c.productSvc.CreateProduct(ctx, product)
		if err != nil {
			log.Printf("[ERROR] Failed to create product %d: %v", printfulProduct.ID, err)
			continue
		}

		successCount++
		job.ItemsProcessed = i + 1
		job.Progress = int((float64(job.ItemsProcessed) / float64(job.ItemsTotal)) * 100)
	}

	completedAt := time.Now()
	job.CompletedAt = &completedAt
	job.Status = "completed"
	job.ItemsProcessed = successCount

	log.Printf("[INFO] Catalog sync completed: %d/%d products synced", successCount, job.ItemsTotal)

	return job, nil
}

// ImportProduct imports a specific Printful product into the shop-api
func (c *PrintfulClient) ImportProduct(ctx context.Context, req *models.PrintfulProductImportRequest) (*models.Product, error) {
	// Parse product ID
	productIDStr := strings.TrimPrefix(req.PrintfulProductID, "printful_")
	productID, err := strconv.Atoi(productIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid product ID: %w", err)
	}

	// Fetch product from Printful
	printfulProduct, err := c.GetProduct(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch product: %w", err)
	}

	// Fetch variants
	variants, err := c.GetProductVariants(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch variants: %w", err)
	}

	// Filter variants if specific variant IDs were requested
	if len(req.VariantIDs) > 0 {
		filteredVariants := []models.PrintfulVariant{}
		for _, variant := range variants {
			variantIDStr := fmt.Sprintf("printful_%d-%d", productID, variant.ID)
			for _, requestedID := range req.VariantIDs {
				if variantIDStr == requestedID || fmt.Sprintf("%d", variant.ID) == requestedID {
					filteredVariants = append(filteredVariants, variant)
					break
				}
			}
		}
		variants = filteredVariants
	}

	// Convert to shop-api product
	productInput, err := c.convertPrintfulProduct(printfulProduct, variants)
	if err != nil {
		return nil, fmt.Errorf("failed to convert product: %w", err)
	}

	// Override title and description if provided
	if req.Title != "" {
		productInput.Title = req.Title
	}
	if req.Description != "" {
		productInput.Description = req.Description
	}

	// Apply markup to prices
	if req.MarkupPercentage > 0 {
		markupMultiplier := 1 + (req.MarkupPercentage / 100)
		productInput.Price = productInput.Price * markupMultiplier
		for i := range productInput.Variants {
			productInput.Variants[i].Price = productInput.Variants[i].Price * markupMultiplier
		}
	}

	// Convert ProductInput to Product for creation
	product := c.productInputToProduct(productInput)

	// Create product in database
	err = c.productSvc.CreateProduct(ctx, product)
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	return product, nil
}

// convertPrintfulProduct converts a Printful product to a shop-api product
func (c *PrintfulClient) convertPrintfulProduct(printfulProduct *models.PrintfulProduct, variants []models.PrintfulVariant) (*models.ProductInput, error) {
	if len(variants) == 0 {
		return nil, fmt.Errorf("product has no variants")
	}

	// Get base price from first variant
	basePrice, err := strconv.ParseFloat(variants[0].Price, 64)
	if err != nil {
		basePrice = 0
	}

	// Convert variants
	productVariants := make([]models.ProductVariantInput, 0, len(variants))
	for _, pv := range variants {
		if !pv.InStock {
			continue
		}

		price, err := strconv.ParseFloat(pv.Price, 64)
		if err != nil {
			price = basePrice
		}

		variant := models.ProductVariantInput{
			SKU:       fmt.Sprintf("PF-%d", pv.ID),
			Title:     pv.Name,
			Price:     price,
			Inventory: 9999, // Printful manages inventory
			Dimensions: pv.Dimensions,
			FulfillmentData: models.FulfillmentData{
				PartnerID:        "printful",
				PartnerProductID: fmt.Sprintf("%d", printfulProduct.ID),
				PartnerVariantID: fmt.Sprintf("%d", pv.ID),
				RequiresShipping: true,
			},
		}

		// Add color and size options
		if pv.Color != "" {
			variant.Options = append(variant.Options, models.VariantOption{
				Name:  "Color",
				Value: pv.Color,
			})
		}
		if pv.Size != "" {
			variant.Options = append(variant.Options, models.VariantOption{
				Name:  "Size",
				Value: pv.Size,
			})
		}

		productVariants = append(productVariants, variant)
	}

	if len(productVariants) == 0 {
		return nil, fmt.Errorf("no in-stock variants available")
	}

	product := &models.ProductInput{
		Title:       printfulProduct.Name,
		Description: printfulProduct.Description,
		Price:       basePrice,
		SKU:         fmt.Sprintf("PF-%d", printfulProduct.ID),
		Status:      "active",
		Inventory:   9999,
		Tags:        []string{printfulProduct.Category, "printful"},
		Variants:    productVariants,
		FulfillmentData: models.FulfillmentData{
			PartnerID:        "printful",
			PartnerProductID: fmt.Sprintf("%d", printfulProduct.ID),
			RequiresShipping: true,
		},
	}

	return product, nil
}

// productInputToProduct converts a ProductInput to a Product
func (c *PrintfulClient) productInputToProduct(input *models.ProductInput) *models.Product {
	// Convert variant inputs to variants
	variants := make([]models.ProductVariant, len(input.Variants))
	for i, v := range input.Variants {
		variants[i] = models.ProductVariant{
			SKU:             v.SKU,
			Title:           v.Title,
			Price:           v.Price,
			Inventory:       v.Inventory,
			Options:         v.Options,
			Dimensions:      v.Dimensions,
			FulfillmentData: v.FulfillmentData,
		}
	}

	return &models.Product{
		Title:           input.Title,
		Description:     input.Description,
		Price:           input.Price,
		SKU:             input.SKU,
		Status:          input.Status,
		Inventory:       input.Inventory,
		Tags:            input.Tags,
		CustomFields:    input.CustomFields,
		Variants:        variants,
		Dimensions:      input.Dimensions,
		FulfillmentData: input.FulfillmentData,
	}
}
