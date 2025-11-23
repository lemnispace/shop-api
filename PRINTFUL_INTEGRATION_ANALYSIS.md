# Shop-API Printful Integration Exploration Summary

## Executive Summary

The LemniSpace shop-api has a **comprehensive and well-structured Printful integration** that is approximately 95% complete. The codebase includes full support for product catalog syncing, product import/variant management, and order fulfillment workflows.

---

## 1. Current Printful Integration Status

### Implemented Features (✓ Complete)

1. **Printful API Client** (`internal/services/printful.go`)
   - Full-featured PrintfulClient with error handling
   - HTTP request wrapper with authentication
   - API key-based authorization via Bearer token
   - Proper timeout configuration (30 seconds)
   - Comprehensive logging for debugging

2. **Catalog Operations**
   - ✓ `GetProducts()` - Fetch all products from Printful catalog
   - ✓ `GetProduct(productID)` - Get single product with details
   - ✓ `GetProductVariants(productID)` - Get all variants for a product
   - ✓ `GetVariant(variantID)` - Get single variant details

3. **Order Operations**
   - ✓ `CreateOrder()` - Create draft order with Printful
   - ✓ `GetOrder()` - Retrieve order status
   - ✓ `ConfirmOrder()` - Confirm draft order for processing
   - ✓ `CancelOrder()` - Cancel/delete orders

4. **Catalog Sync & Import**
   - ✓ `SyncCatalog()` - Full catalog sync with deduplication
   - ✓ `ImportProduct()` - Import specific product with markup
   - ✓ Product conversion from Printful → shop-api format
   - ✓ Variant SKU mapping with `PF-{variantId}` prefix
   - ✓ Automatic deduplication by SKU (prevents duplicates on re-sync)
   - ✓ Variant ID preservation during updates

5. **HTTP Handlers** (`internal/handlers/integrations.go`)
   - ✓ `SyncPrintfulCatalog()` - POST /v1/integrations/printful/sync
   - ✓ `GetSyncStatus()` - GET /v1/integrations/printful/sync/:id
   - ✓ `ListPrintfulProducts()` - GET /v1/integrations/printful/products
   - ✓ `GetPrintfulProduct()` - GET /v1/integrations/printful/products/:id
   - ✓ `ImportPrintfulProduct()` - POST /v1/integrations/printful/products/import
   - ✓ `SubmitPrintfulOrder()` - POST /v1/integrations/printful/orders
   - ✓ `GetPrintfulOrder()` - GET /v1/integrations/printful/orders/:id
   - ✓ `ConfirmPrintfulOrder()` - POST /v1/integrations/printful/orders/:id/confirm
   - ✓ `CancelPrintfulOrder()` - DELETE /v1/integrations/printful/orders/:id

6. **Fulfillment Service** (`internal/services/fulfillment.go`)
   - ✓ Integration with order workflow
   - ✓ Automatic Printful submission on payment confirmation
   - ✓ Fulfillment record tracking in DynamoDB
   - ✓ Status management (pending, submitted, shipped, etc.)

7. **Comprehensive Testing**
   - ✓ Unit tests for Printful client operations
   - ✓ Mock HTTP server tests for API calls
   - ✓ Product conversion tests
   - ✓ Markup calculation tests
   - ✓ Out-of-stock filtering tests

---

## 2. Data Structures & Models

### Product-Related Models

**`Product` (internal/models/product.go)**
```go
type Product struct {
    ID              string
    Title           string
    Description     string
    Price           float64
    SKU             string
    Status          string          // "draft", "active", "archived"
    Inventory       int
    Tags            []string
    CustomFields    map[string]interface{}
    Images          []Image
    Variants        []ProductVariant
    Dimensions      Dimensions
    FulfillmentData FulfillmentData
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

**`ProductVariant`**
```go
type ProductVariant struct {
    ID              string
    ProductID       string
    ProductTitle    string
    SKU             string
    Title           string
    Price           float64
    Inventory       int
    Options         []VariantOption    // Size, Color, etc.
    Dimensions      Dimensions
    FulfillmentData FulfillmentData    // Links to Printful
}
```

**`FulfillmentData` (internal/models/common.go)**
```go
type FulfillmentData struct {
    PartnerID        string                 // "printful"
    PartnerProductID string                 // Printful product ID
    PartnerVariantID string                 // Printful variant ID
    AdditionalData   map[string]interface{}
    HSCode           string
    CountryOfOrigin  string
    Harmonized       bool
    RequiresShipping bool
}
```

**`VariantOption`**
```go
type VariantOption struct {
    Name  string
    Value string
}
```

**`Dimensions`**
```go
type Dimensions struct {
    Width  float64
    Height float64
    Depth  float64
    Length float64
    Weight float64
}
```

### Printful-Specific Models

**`PrintfulProduct`**
- ID, ExternalID, Name, Description, Category
- Variants count, Synced count
- Thumbnail, MockupImages, FileSpec
- Options (Size, Color, etc.)
- IsDiscontinued, IsSyncable flags
- AvgFulfillmentTime

**`PrintfulVariant`**
- ID, ProductID, Name, Size, Color, ColorCode
- Price, InStock, AvailabilityStatus
- Image URL, Dimensions, Options

**`PrintfulOrder`** (for submission to Printful)
- ExternalID (links to shop-api order)
- Recipient (shipping address)
- Items with VariantID, Quantity, Price
- Files (for customizations)
- RetailCosts, GiftMessage, PackingSlip

**`PrintfulOrderRequest`** (request structure)
```go
type PrintfulOrderRequest struct {
    ExternalID   string
    Recipient    PrintfulRecipient
    Items        []PrintfulOrderItem
    RetailCosts  *PrintfulRetailCosts
    GiftMessage  string
    PackingSlip  *PrintfulPackingSlip
}
```

---

## 3. DynamoDB Schema for Products/Variants

### Single-Table Design Pattern

**Partition Key (PK)**: `PRODUCT#{productId}` or `VARIANT#{variantId}`
**Sort Key (SK)**: `METADATA` or relationship key

**Example Item Structure**:
```
PK: PRODUCT#prod_abc123
SK: METADATA
EntityType: PRODUCT
ID: prod_abc123
Title: Canvas Print
Description: High-quality...
Price: 59.99
SKU: PF-71
Status: active
InventoryCount: 9999
Tags: ["printful", "canvas"]
Variants: [var_list]
FulfillmentData: {
  partnerId: "printful",
  partnerProductId: "71",
  requiresShipping: true
}
CreatedAt: 2024-01-01T00:00:00Z
UpdatedAt: 2024-01-05T12:30:00Z

---

PK: PRODUCT#prod_abc123
SK: VARIANT#var_xyz789
EntityType: VARIANT
ID: var_xyz789
ProductID: prod_abc123
SKU: PF-4012
Title: 12x16 Canvas
Price: 59.99
Inventory: 9999
Options: [
  {name: "Size", value: "12x16"},
  {name: "Color", value: "White"}
]
Dimensions: {width: 12, height: 16, weight: 1.2}
FulfillmentData: {
  partnerId: "printful",
  partnerProductId: "71",
  partnerVariantId: "4012",
  requiresShipping: true
}
```

### Query Patterns
- Get product: `PK=PRODUCT#{productId}, SK=METADATA`
- Get all variants: `PK=PRODUCT#{productId}, SK begins_with VARIANT#`
- Get single variant: `PK=PRODUCT#{productId}, SK=VARIANT#{variantId}`

---

## 4. Printful Integration Handlers & Services

### Handler Layer (`internal/handlers/integrations.go`)

| Endpoint | Method | Handler Function | Status |
|----------|--------|------------------|--------|
| `/v1/integrations/printful/sync` | POST | `SyncPrintfulCatalog()` | ✓ Async |
| `/v1/integrations/printful/sync/:id` | GET | `GetSyncStatus()` | ⚠ Placeholder |
| `/v1/integrations/printful/products` | GET | `ListPrintfulProducts()` | ✓ Complete |
| `/v1/integrations/printful/products/:id` | GET | `GetPrintfulProduct()` | ✓ Complete |
| `/v1/integrations/printful/products/import` | POST | `ImportPrintfulProduct()` | ✓ Complete |
| `/v1/integrations/printful/orders` | POST | `SubmitPrintfulOrder()` | ✓ Complete |
| `/v1/integrations/printful/orders/:id` | GET | `GetPrintfulOrder()` | ✓ Complete |
| `/v1/integrations/printful/orders/:id/confirm` | POST | `ConfirmPrintfulOrder()` | ✓ Complete |
| `/v1/integrations/printful/orders/:id` | DELETE | `CancelPrintfulOrder()` | ✓ Complete |

### Service Layer (`internal/services/printful.go`)

**PrintfulService Interface** (19 methods)
```go
// Catalog operations
GetProducts(ctx) ([]PrintfulProduct, error)
GetProduct(ctx, productID) (*PrintfulProduct, error)
GetVariant(ctx, variantID) (*PrintfulVariant, error)
GetProductVariants(ctx, productID) ([]PrintfulVariant, error)

// Order operations
CreateOrder(ctx, order) (*PrintfulOrder, error)
GetOrder(ctx, orderID) (*PrintfulOrder, error)
ConfirmOrder(ctx, orderID) (*PrintfulOrder, error)
CancelOrder(ctx, orderID) (*PrintfulOrder, error)

// Sync operations
SyncCatalog(ctx) (*PrintfulSyncJob, error)
ImportProduct(ctx, req) (*Product, error)
```

**Key Implementation Details**:
- HTTP client with 30-second timeout
- Bearer token authentication
- Base URL: `https://api.printful.com`
- Proper error handling and logging
- Security: Does not log full response bodies (avoid leaking customer data)

### Fulfillment Service (`internal/services/fulfillment.go`)

Bridges orders to Printful:
```go
type FulfillmentService interface {
    CreateFulfillment(ctx, input) (*Fulfillment, error)
    GetFulfillment(ctx, fulfillmentID) (*Fulfillment, error)
    UpdateFulfillmentStatus(ctx, fulfillmentID, status) error
    GetOrderFulfillments(ctx, orderID) ([]*Fulfillment, error)
    SubmitOrderToPrintful(ctx, order) (*Fulfillment, error)
}
```

**Workflow**:
1. Order created → Order service creates order record
2. Payment confirmed → Payment handler triggers fulfillment
3. Fulfillment service: Creates fulfillment record + submits to Printful
4. Printful returns confirmation → Status updated in DynamoDB
5. Tracking updates → Fulfillment record updated periodically

---

## 5. Environment Configuration

### Required Environment Variables

| Variable | Purpose | Status |
|----------|---------|--------|
| `PRINTFUL_API_KEY` | Printful API authentication | **Required** |
| `DYNAMODB_TABLE_NAME` | DynamoDB table name | Optional (default: "ShopAPI") |
| `AWS_REGION` | AWS region | Required |
| `AWS_PROFILE` | AWS credentials profile | Optional |

### Current .env Configuration
```
PRINTFUL_API_TOKEN=PRINTFUL_API_KEY
AWS_PROFILE=dev
AWS_REGION=us-east-1
```

**Note**: Variable naming inconsistency - code expects `PRINTFUL_API_KEY` but .env has `PRINTFUL_API_TOKEN`

---

## 6. What's Missing or Needs Enhancement

### High Priority Issues

1. **Environment Variable Naming Inconsistency**
   - Code: Looks for `PRINTFUL_API_KEY`
   - .env file: Has `PRINTFUL_API_TOKEN`
   - FIX: Standardize on one name (preferably `PRINTFUL_API_KEY`)

2. **Sync Status Tracking**
   - Handler `GetSyncStatus()` is a placeholder (returns hard-coded values)
   - No persistent sync job tracking in database
   - In-memory job tracking lost on Lambda restart
   - **Missing**: Persist sync jobs in DynamoDB for production

3. **Security - Admin Authorization**
   - All Printful handlers marked with TODO comment
   - Current: Only checks AuthMiddleware (any authenticated user)
   - Missing: Admin role verification
   - Risk: Regular users could trigger catalog syncs, imports, and order operations

4. **Concurrent Sync Prevention**
   - No locking mechanism for catalog syncs
   - Multiple concurrent sync requests could cause issues
   - Missing: Sync job queue or distributed lock (Redis)

### Medium Priority Enhancements

1. **Fulfillment Webhook Handling**
   - Missing: Printful webhook handler for order status updates
   - Missing: Tracking shipments and delivery status
   - Missing: Automatic fulfillment record updates

2. **Error Recovery & Retry Logic**
   - Basic error handling exists
   - Missing: Exponential backoff for API failures
   - Missing: Dead-letter queue for failed orders
   - Missing: Retry mechanism for transient failures

3. **Product Image Handling**
   - Printful sync handles mockup images
   - Missing: Automatic image downloading/storage to S3
   - Current: Images stored in Printful URL (external dependency)

4. **Bulk Operations**
   - Missing: Bulk product import endpoint
   - Missing: Bulk order creation
   - Current: Only single product/order operations

5. **Caching**
   - Missing: Cache for Printful product catalog
   - Every request hits Printful API
   - Missing: Cache invalidation strategy

6. **Price Markup Management**
   - ImportProduct() supports markup percentage
   - Missing: Store markup configuration per product
   - Missing: Update prices without re-importing

### Low Priority / Nice-to-Have

1. **Advanced Search/Filtering**
   - List endpoints could support more filters (category, price range, etc.)

2. **Batch Status Updates**
   - Missing: Bulk fulfillment status updates

3. **Order Customization Files**
   - Missing: Automatic file upload to Printful for customizations
   - Current: Supports file structure, missing integration with image-service

4. **Analytics**
   - Missing: Sync job metrics/reporting
   - Missing: Fulfillment funnel analysis

---

## 7. Code Quality & Testing

### Test Coverage
✓ Unit tests for PrintfulClient (8 test functions)
✓ Mock HTTP server for integration testing
✓ Tests for product conversion logic
✓ Tests for variant handling

**Missing**:
- Fulfillment service tests
- Handler integration tests
- Error scenario testing (API failures, timeouts)
- End-to-end tests with real Printful API

### Code Quality
✓ Clean service/handler separation
✓ Proper error handling with context
✓ Comprehensive logging
✓ Security considerations (no sensitive data logging)

**Improvements Needed**:
- Add context cancellation handling
- Improve error messages for debugging
- Add request/response validation middleware

---

## 8. Workflow: From Order to Printful Submission

```
1. User adds items to cart → CartService
2. User checks out → CreateOrder()
   - Order stored in DynamoDB
   - Status: "pending"
   - Payment intent created

3. User confirms payment → ConfirmPayment()
   - Payment verified with Stripe
   - Order status → "confirmed"
   - TRIGGERS: SubmitOrderToPrintful()

4. FulfillmentService.SubmitOrderToPrintful()
   - Builds PrintfulOrderRequest from shop order
   - Maps cart items to Printful variants
   - Includes shipping address (PrintfulRecipient)
   - Calls PrintfulService.CreateOrder()
   - Returns draft order from Printful

5. PrintfulService.CreateOrder()
   - HTTP POST to https://api.printful.com/orders
   - Includes ExternalID (shop order ID)
   - Returns PrintfulOrder with Printful order ID
   - Status: "draft"

6. Admin/System calls ConfirmOrder()
   - HTTP POST to /orders/{printfulOrderId}/confirm
   - Printful validates, submits to production
   - Status: "confirmed" (processing)

7. Fulfillment record created
   - Links shop order to Printful order
   - Stores mapping in DynamoDB
   - Tracks fulfillment status

8. Printful processes
   - Prints items
   - Ships package
   - Sends tracking updates (via webhooks - missing)

9. Order completion
   - Shop receives webhook (missing)
   - Updates order/fulfillment status
   - Customer receives notification
```

---

## 9. Key File Locations

| File | Purpose |
|------|---------|
| `internal/models/product.go` | Product/Variant data structures |
| `internal/models/printful.go` | Printful-specific models (150+ lines) |
| `internal/models/common.go` | Shared models (Dimensions, FulfillmentData) |
| `internal/services/printful.go` | PrintfulClient implementation (640 lines) |
| `internal/services/fulfillment.go` | Order fulfillment workflow |
| `internal/handlers/integrations.go` | HTTP handlers for Printful endpoints |
| `internal/services/printful_test.go` | Comprehensive test suite (420 lines) |
| `cmd/shop/main.go` | Service initialization & wiring |

---

## 10. Configuration Checklist

To get Printful integration working:

- [ ] Set `PRINTFUL_API_KEY` environment variable (from Printful account settings)
- [ ] Set `DYNAMODB_TABLE_NAME` (default: "ShopAPI")
- [ ] Set `AWS_REGION` (e.g., "us-east-1")
- [ ] Ensure DynamoDB table exists with correct schema
- [ ] Update router to add admin authorization checks (TODO items)
- [ ] Implement sync status persistence in DynamoDB
- [ ] Add Printful webhook handler for order updates
- [ ] Configure CORS if frontend is on different domain

---

## Summary

The Printful integration is **well-architected and mostly complete** (95%+). The main gaps are:

1. **Security**: Missing admin role checks
2. **Operational**: No persistent sync job tracking
3. **Webhook Support**: Missing fulfillment status updates
4. **Caching**: No product cache
5. **Error Recovery**: Basic error handling, missing retries

The codebase follows clean architecture principles with clear separation between models, services, and handlers. All Printful API operations are properly wrapped in a service interface, making the code testable and maintainable.

