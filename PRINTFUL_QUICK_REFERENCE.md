# Printful Integration Quick Reference

## Current State: 95% Complete

All core functionality is implemented. Main gaps are operational/security enhancements.

---

## Key Files to Know

```
internal/models/
  ├── product.go          # Product, ProductVariant structures
  ├── printful.go         # 50+ Printful models (PrintfulProduct, PrintfulOrder, etc.)
  └── common.go           # Shared: FulfillmentData, Dimensions, VariantOption

internal/services/
  ├── printful.go         # PrintfulClient (19 API methods)
  ├── printful_test.go    # 8 comprehensive tests
  └── fulfillment.go      # Order → Printful workflow

internal/handlers/
  └── integrations.go     # 9 HTTP handlers for Printful endpoints

cmd/shop/
  └── main.go             # Service wiring (lines 100-119)
```

---

## API Endpoints (All Implemented)

### Catalog
- `GET /v1/integrations/printful/products` - List products
- `GET /v1/integrations/printful/products/{id}` - Get product details
- `POST /v1/integrations/printful/sync` - Start catalog sync
- `GET /v1/integrations/printful/sync/{id}` - Check sync status

### Import
- `POST /v1/integrations/printful/products/import` - Import single product

### Orders
- `POST /v1/integrations/printful/orders` - Create order
- `GET /v1/integrations/printful/orders/{id}` - Get order status
- `POST /v1/integrations/printful/orders/{id}/confirm` - Confirm order
- `DELETE /v1/integrations/printful/orders/{id}` - Cancel order

---

## Critical Issues to Fix

### 1. Environment Variable Mismatch
**File**: `/Users/santiagogomez/Projects/LemniSpace/shop-api/.env`

```
Current:   PRINTFUL_API_TOKEN=...
Required:  PRINTFUL_API_KEY=...
```

**Why**: Code at `cmd/shop/main.go:101` looks for `PRINTFUL_API_KEY`

**Fix**: 
```bash
# Rename in .env
mv PRINTFUL_API_TOKEN PRINTFUL_API_KEY
```

### 2. Missing Admin Authorization
**File**: `internal/handlers/integrations.go` (lines 22-24, marked with TODO)

```go
// TODO(security): Enforce an explicit admin/ops authorization check 
// before allowing access to Printful integration endpoints
```

**Risk**: Any authenticated user can trigger syncs, imports, order operations

**Fix**: Add admin middleware check
```go
protected := integrations.Group("/printful")
protected.Use(middleware.IsAdminMiddleware)  // ADD THIS
{
    protected.POST("/sync", handlers.SyncPrintfulCatalog)
    // ... other endpoints
}
```

### 3. Sync Status Tracking is Placeholder
**File**: `internal/handlers/integrations.go:65-79`

Current implementation returns hard-coded values. Loses state on Lambda restart.

**Fix**: Implement in DynamoDB
- Store sync jobs in DynamoDB with `SYNCREQUEST#` prefix
- Return actual status instead of placeholders
- Support job history/filtering

### 4. Concurrent Sync Prevention Missing
**File**: `internal/handlers/integrations.go:26-63`

Multiple concurrent sync requests could corrupt data.

**Fix**: Add distributed lock
- Option 1: Redis lock (add `REDIS_URL` env var)
- Option 2: DynamoDB conditional write
- Option 3: Simple in-memory lock (not suitable for Lambda)

---

## Product Integration Points

### When Product is Imported from Printful

The product structure maps Printful fields to shop-api:

```
Printful Field          →  Shop-API Field
ProductID (int)         →  SKU: "PF-{productID}"
Product.Name            →  title
Product.Description     →  description
Product.Category        →  tags
VariantID (int)         →  SKU: "PF-{variantID}"
Variant.Price (string)  →  price (float64)
Variant.Color           →  options[Color]
Variant.Size            →  options[Size]
```

**Conversion Logic**: `internal/services/printful.go:527-603`

### DynamoDB Storage

```
PK: PRODUCT#prod_abc123
SK: METADATA
FulfillmentData: {
  partnerId: "printful"
  partnerProductId: "71"         // Printful product ID
  partnerVariantId: "4012"       // Printful variant ID
  requiresShipping: true
}

---

PK: PRODUCT#prod_abc123
SK: VARIANT#var_xyz789
FulfillmentData: {
  partnerId: "printful"
  partnerProductId: "71"
  partnerVariantId: "4012"
  requiresShipping: true
}
```

---

## Order Fulfillment Flow

```
User Order → Payment → Fulfillment → Printful → Shipped
   ↓          ↓           ↓            ↓         ↓
Order.Create Payment  SubmitOrder   CreateOrder Track
           Confirm   ToPrintful     Response    Update
                         ↓
                   Fulfillment
                   Record Created
                   in DynamoDB
```

**Key Code**:
- Order creation: `internal/services/order.go`
- Payment handling: `internal/handlers/payment.go`
- Fulfillment: `internal/services/fulfillment.go:195-250` (SubmitOrderToPrintful)
- Printful submission: `internal/services/printful.go:226-244` (CreateOrder)

---

## Testing the Integration

### Prerequisites
```bash
# 1. Set environment variable
export PRINTFUL_API_KEY="your-api-key"

# 2. Ensure DynamoDB is running
make dynamo-local

# 3. Initialize tables
make dynamo-init
```

### Test Endpoints
```bash
# 1. List Printful products
curl -H "Authorization: Bearer $(jwt-token)" \
  http://localhost:8080/v1/integrations/printful/products

# 2. Import a product
curl -X POST http://localhost:8080/v1/integrations/printful/products/import \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $(jwt-token)" \
  -d '{
    "printfulProductId": "71",
    "markupPercentage": 30,
    "title": "Custom Canvas"
  }'

# 3. Sync entire catalog (async)
curl -X POST http://localhost:8080/v1/integrations/printful/sync \
  -H "Authorization: Bearer $(jwt-token)"
```

### Unit Tests
```bash
cd /Users/santiagogomez/Projects/LemniSpace/shop-api
go test ./internal/services -run TestPrintful -v
```

---

## Common Printful Data Structures

### PrintfulVariant (from Printful API)
- `ID` (int): Printful variant ID
- `ProductID` (int): Parent product ID
- `Name` (string): Display name (e.g., "12x16 Canvas")
- `Size` (string): Size option
- `Color` (string): Color name
- `ColorCode` (string): Hex code
- `Price` (string): Cost price as string
- `InStock` (bool): Inventory status
- `Image` (string): Preview image URL

### PrintfulOrderRequest (to submit)
```json
{
  "external_id": "order_abc123",
  "recipient": {
    "name": "John Doe",
    "address1": "123 Main St",
    "city": "New York",
    "state_code": "NY",
    "country_code": "US",
    "zip": "10001"
  },
  "items": [
    {
      "sync_variant_id": 4012,
      "quantity": 1,
      "price": "59.99"
    }
  ]
}
```

### PrintfulOrder (response)
- `ID` (int): Printful order ID
- `ExternalID` (string): Links to shop order
- `Status` (string): "draft", "confirmed", "processing", "shipped", etc.
- `Items` ([]PrintfulOrderItem): Order line items
- `Shipments` ([]PrintfulShipment): Tracking info (when shipped)

---

## Environment Variables Checklist

| Variable | Example | Required | Notes |
|----------|---------|----------|-------|
| `PRINTFUL_API_KEY` | `7Ad82lmVht0QhKIZMCEwU...` | YES | From Printful account settings |
| `DYNAMODB_TABLE` | `ShopAPI` | NO | Defaults to "ShopAPI" |
| `DYNAMODB_ENDPOINT` | `http://localhost:8000` | NO | For local dev only |
| `AWS_REGION` | `us-east-1` | YES | AWS region |
| `AWS_PROFILE` | `default` | NO | For local dev |
| `RUN_LOCAL` | `true` | NO | Set to run locally |

---

## Service Injection (main.go)

Printful service is initialized at startup (line 100-109):

```go
printfulAPIKey := os.Getenv("PRINTFUL_API_KEY")
if printfulAPIKey == "" {
    log.Printf("WARNING: PRINTFUL_API_KEY not set...")
} else {
    printfulService = services.NewPrintfulClient(printfulAPIKey, productService)
    handlers.SetPrintfulService(printfulService)
}
```

Then injected into handlers via `SetPrintfulService()`.

---

## Next Steps to Production

1. [ ] Fix environment variable naming (`PRINTFUL_API_TOKEN` → `PRINTFUL_API_KEY`)
2. [ ] Add admin authorization middleware to Printful endpoints
3. [ ] Implement persistent sync job tracking in DynamoDB
4. [ ] Add Printful webhook handler for order status updates
5. [ ] Implement retry logic with exponential backoff
6. [ ] Add product caching with TTL
7. [ ] Write fulfillment service tests
8. [ ] Test end-to-end with real Printful account
9. [ ] Configure CloudWatch logging
10. [ ] Performance test concurrent syncs

---

## Useful References

- **Printful API Docs**: https://www.printful.com/api/docs/
- **Printful Product Catalog**: https://dashboard.printful.com/products
- **Shop API Design**: `API_DESIGN.md`
- **Implementation Status**: `API_IMPLEMENTATION_STATUS.md`
