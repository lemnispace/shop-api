# Printful Integration Architecture

## System Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         Web Client                              │
│                    (Next.js Frontend)                           │
└─────────────────────────────┬───────────────────────────────────┘
                              │
                              │ HTTP/REST
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    API Gateway                                  │
│               (AWS API Gateway v1)                              │
├─────────────────────────────────────────────────────────────────┤
│                 /v1/integrations/printful/*                     │
│    /v1/products/*   /v1/orders/*   /v1/customizations/*         │
└─────────────────────────────────┬───────────────────────────────┘
                                  │
                                  ▼
        ┌─────────────────────────────────────────┐
        │     Gin Router                          │
        │  internal/routers/router.go             │
        │  • Route handlers                       │
        │  • Middleware chain                     │
        │  • Auth enforcement                     │
        └──────────────────┬──────────────────────┘
                           │
        ┌──────────────────┴──────────────────┐
        │                                     │
        ▼                                     ▼
┌──────────────────────┐          ┌──────────────────────┐
│  HTTP Handlers       │          │  Middleware          │
│  handlers/           │          │  middleware/         │
│  • products.go       │          │  • auth.go           │
│  • orders.go         │          │  • cors.go           │
│  • integrations.go   │          │  • logging.go        │
│    ├── Sync          │          │  • validation.go     │
│    ├── Import        │          └──────────────────────┘
│    ├── Orders        │
│    └── Status        │
└──────────┬───────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────────────┐
│                    SERVICE LAYER                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────────────────┐    ┌─────────────────────┐   │
│  │ PrintfulService              │    │ ProductService      │   │
│  │ internal/services/printful.go│    │ internal/services   │   │
│  │                              │    │ • CreateProduct()   │   │
│  │ • GetProducts()              │◄───┤ • UpdateProduct()   │   │
│  │ • GetProductVariants()       │    │ • GetProduct()      │   │
│  │ • GetVariant()               │    │ • ListProducts()    │   │
│  │ • SyncCatalog()              │    │ • Variants ops      │   │
│  │ • ImportProduct()            │    └─────────────────────┘   │
│  │ • CreateOrder()              │                               │
│  │ • GetOrder()                 │    ┌─────────────────────┐   │
│  │ • ConfirmOrder()             │    │ FulfillmentService  │   │
│  │ • CancelOrder()              │    │ internal/services   │   │
│  └──────────────┬───────────────┘    │ fulfillment.go      │   │
│                 │                    │                     │   │
│                 │  Uses              │ • CreateFulfillment │   │
│                 ▼                    │ • SubmitOrderTo     │   │
│         ┌──────────────────┐         │   Printful()        │   │
│         │ HTTP Client      │         │ • UpdateStatus()    │   │
│         │ makeRequest()    │         │ • GetFulfillment()  │   │
│         │                  │         └─────────────────────┘   │
│         │ Bearer Auth      │                                    │
│         │ Timeout: 30s     │         ┌─────────────────────┐   │
│         │ Base URL: ...    │         │ Other Services      │   │
│         └──────────────────┘         │ • CartService       │   │
│                                      │ • OrderService      │   │
│                                      │ • CustomerService   │   │
└──────────────────────────────────────┴─────────────────────┘   │
                                                                   │
└─────────────────────────────────────────────────────────────────┘
           │                                    │
           │                                    │
           ▼                                    ▼
    ┌─────────────────┐              ┌──────────────────┐
    │   Printful API  │              │    DynamoDB      │
    │ api.printful.com│              │  Single Table    │
    │                 │              │  Design          │
    │ • Products      │              │                  │
    │ • Variants      │              │  PK: PRODUCT#... │
    │ • Orders        │              │  PK: ORDER#...   │
    │ • Fulfillment   │              │  PK: FULFILLMENT │
    └─────────────────┘              │  ...             │
                                     └──────────────────┘
```

---

## Data Flow: Product Import

```
┌─────────────────────┐
│ Admin clicks Import │
└──────┬──────────────┘
       │
       ▼
POST /v1/integrations/printful/products/import
{
  "printfulProductId": "71",
  "markupPercentage": 30,
  "title": "Custom Canvas"
}
       │
       ▼
┌──────────────────────────────────┐
│ ImportPrintfulProduct Handler    │
│ handlers/integrations.go:230     │
└──────────────┬───────────────────┘
               │
               ▼
┌──────────────────────────────────┐
│ PrintfulService.ImportProduct()  │
│ services/printful.go:417         │
│                                  │
│ 1. GetProduct(productID)         │
│ 2. GetProductVariants(productID) │
│ 3. convertPrintfulProduct()      │
│ 4. Check for duplicates by SKU   │
│ 5. Create or Update Product      │
└──────────────┬───────────────────┘
               │
               ├─── HTTP Request ──► Printful API
               │                    /products/{id}
               │                    /products/{id}/variants
               │
               └─► Response:
                   - ProductID, Name, Description
                   - Variants with Size, Color, Price
                   - Mockup Images, Dimensions
               │
               ▼
┌──────────────────────────────────┐
│ ProductService.CreateProduct()   │
│ OR                               │
│ ProductService.UpdateProduct()   │
│                                  │
│ - Generate Product ID            │
│ - Build FulfillmentData:         │
│   {                              │
│     partnerId: "printful"        │
│     partnerProductId: "71"       │
│     partnerVariantId: "4012"     │
│   }                              │
│ - Generate Variant IDs           │
│ - Build Variant Objects          │
└──────────────┬───────────────────┘
               │
               ▼
┌──────────────────────────────────┐
│ DynamoDB: PutItem                │
│                                  │
│ Item 1:                          │
│  PK: PRODUCT#prod_abc123         │
│  SK: METADATA                    │
│  {title, price, sku, etc}        │
│  FulfillmentData: {...}          │
│                                  │
│ Items 2-N:                       │
│  PK: PRODUCT#prod_abc123         │
│  SK: VARIANT#var_xyz789          │
│  {title, price, options, etc}    │
│  FulfillmentData: {...}          │
└──────────────────────────────────┘
               │
               ▼
        ✓ Product Imported
```

---

## Data Flow: Order to Printful Submission

```
┌──────────────────┐
│ Customer Places  │
│      Order       │
└──────┬───────────┘
       │
       ▼
POST /v1/orders
{
  "customerId": "cust_123",
  "cartId": "cart_456",
  "items": [...]
}
       │
       ▼
┌──────────────────────────────────┐
│ CreateOrder Handler              │
│ handlers/orders.go               │
└──────────────┬───────────────────┘
               │
               ▼
┌──────────────────────────────────┐
│ OrderService.CreateOrder()       │
│ • Generate Order ID              │
│ • Save to DynamoDB               │
│ • Return Order with Status:      │
│   "pending"                      │
└──────────────┬───────────────────┘
               │
               ▼
┌──────────────────────────────────┐
│ Customer Confirms Payment        │
│ /orders/{id}/confirm-payment     │
└──────────────┬───────────────────┘
               │
               ▼
┌──────────────────────────────────┐
│ ConfirmPayment Handler           │
│ handlers/payment.go              │
│ • Verify Stripe webhook          │
│ • Update Order status:           │
│   "confirmed"                    │
└──────────────┬───────────────────┘
               │
               ▼
┌──────────────────────────────────┐
│ FulfillmentService.              │
│ SubmitOrderToPrintful()          │
│ services/fulfillment.go:195      │
│                                  │
│ 1. Build PrintfulOrderRequest:   │
│    {                             │
│      external_id: "ord_xyz",     │
│      recipient: {...},           │
│      items: [                    │
│        {sync_variant_id: 4012,   │
│         quantity: 1,             │
│         price: "59.99"}          │
│      ]                           │
│    }                             │
│ 2. Call PrintfulService.         │
│    CreateOrder()                 │
└──────────────┬───────────────────┘
               │
               ▼
┌──────────────────────────────────┐
│ PrintfulService.CreateOrder()    │
│ services/printful.go:226         │
│                                  │
│ POST https://api.printful.com/   │
│     orders                       │
│                                  │
│ Headers:                         │
│  Authorization: Bearer {key}     │
│  Content-Type: application/json  │
└──────────────┬───────────────────┘
               │
               ▼
        Printful API Response:
        {
          "code": 200,
          "result": {
            "id": 123456,
            "status": "draft",
            "external_id": "ord_xyz"
          }
        }
               │
               ▼
┌──────────────────────────────────┐
│ FulfillmentService.              │
│ CreateFulfillment()              │
│ services/fulfillment.go:52       │
│                                  │
│ • Generate Fulfillment ID        │
│ • Store mapping:                 │
│   orderId → printfulOrderId      │
│ • Status: "pending"              │
└──────────────┬───────────────────┘
               │
               ▼
┌──────────────────────────────────┐
│ DynamoDB: PutItem                │
│                                  │
│ PK: FULFILLMENT#fulf_abc123      │
│ SK: METADATA                     │
│ {                                │
│   orderId: "ord_xyz",            │
│   printfulOrderId: 123456,       │
│   status: "pending",             │
│   items: [...]                   │
│ }                                │
└──────────────────────────────────┘
               │
               ▼
        ✓ Fulfillment Created
          Order sent to Printful
```

---

## Data Model: Product & Variant in DynamoDB

```
Product Item:
─────────────────────────────────
PK: PRODUCT#prod_abc123
SK: METADATA
GSI1PK: COLLECTION#coll_xyz789
GSI1SK: PRODUCT#prod_abc123

Attributes:
  ID: "prod_abc123"
  Title: "Canvas Print"
  Description: "High quality canvas printing"
  Price: 59.99
  SKU: "PF-71"
  Status: "active"
  Inventory: 9999
  Tags: ["printful", "canvas", "wall-art"]
  Images: [
    { id: "img_1", url: "https://...", altText: "" }
  ]
  Dimensions: {
    width: 12.0,
    height: 16.0,
    depth: 0.75,
    weight: 1.2
  }
  FulfillmentData: {
    partnerId: "printful"
    partnerProductId: "71"
    partnerVariantId: null
    requiresShipping: true
  }
  CreatedAt: "2024-01-01T00:00:00Z"
  UpdatedAt: "2024-01-05T12:30:00Z"


Variant Items (multiple per product):
───────────────────────────────────────
PK: PRODUCT#prod_abc123
SK: VARIANT#var_xyz789
GSI1PK: PRODUCT#prod_abc123
GSI1SK: VARIANT#var_xyz789

Attributes:
  ID: "var_xyz789"
  ProductID: "prod_abc123"
  ProductTitle: "Canvas Print"
  Title: "12x16 Canvas"
  Price: 59.99
  SKU: "PF-4012"
  Inventory: 9999
  Options: [
    { name: "Size", value: "12x16" },
    { name: "Color", value: "White" }
  ]
  Dimensions: {
    width: 12.0,
    height: 16.0,
    depth: 0.75,
    weight: 1.2
  }
  FulfillmentData: {
    partnerId: "printful"
    partnerProductId: "71"
    partnerVariantId: "4012"
    requiresShipping: true
  }
```

---

## Service Dependencies

```
                  ┌─────────────────────┐
                  │  Handler Layer      │
                  │  (HTTP Handlers)    │
                  └──────────┬──────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
              ▼              ▼              ▼
        ┌──────────┐  ┌──────────┐  ┌──────────────┐
        │ Printful │  │ Product  │  │ Fulfillment  │
        │ Service  │  │ Service  │  │ Service      │
        └────┬─────┘  └────┬─────┘  └────┬─────────┘
             │             │             │
             │    Uses     │             │
             └─────────────┼─────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │ DynamoDB     │
                    │ Single Table │
                    └──────────────┘
```

**Dependencies**:
- PrintfulService → uses HTTP client (standalone)
- ProductService → uses DynamoDB
- FulfillmentService → uses DynamoDB + PrintfulService
- Handlers → use all services above

---

## Configuration & Initialization

```
cmd/shop/main.go:initServices()
│
├─► Load AWS Config
│   └─► Create DynamoDB Client
│
├─► Initialize ProductService
│   └─► handlers.SetProductService()
│
├─► Initialize CollectionService
│   └─► handlers.SetCollectionService()
│
├─► Initialize CartService
│   └─► handlers.SetCartService()
│
├─► Initialize OrderService
│   └─► handlers.SetOrderService()
│
├─► Initialize PaymentService
│   └─► handlers.SetPaymentService()
│
├─► Initialize CustomizationService
│   └─► handlers.SetCustomizationService()
│
├─► Initialize PrintfulService ◄─── PRINTFUL_API_KEY env var
│   └─► handlers.SetPrintfulService()
│
├─► Initialize FulfillmentService
│   └─► handlers.SetFulfillmentService()
│
├─► Initialize CustomerService
│   └─► handlers.SetCustomerService()
│
└─► Initialize AuthService
    └─► handlers.SetAuthService()

Then: router.InitRouter(authService)
      └─► Register all routes with handlers
```

---

## API Endpoint Tree

```
/v1
├── /products
│   ├── GET      (List - public)
│   ├── POST     (Create - admin)
│   ├── /:id
│   │   ├── GET  (Get - public)
│   │   ├── PUT  (Update - admin)
│   │   ├── DELETE (Delete - admin)
│   │   ├── /variants
│   │   │   ├── GET (List variants - public)
│   │   │   ├── POST (Create - admin)
│   │   │   ├── /:variantId
│   │   │   │   ├── PUT (Update - admin)
│   │   │   │   ├── DELETE (Delete - admin)
│   │   │   │   └── /images
│   │   │   │       └── POST (Associate - admin)
│   │   └── /images
│   │       └── POST (Upload - admin)
│   └── /count
│       └── GET (Count - public)
│
├── /orders
│   ├── POST (Create - auth)
│   ├── GET  (List - auth)
│   ├── /:id
│   │   ├── GET (Get - auth)
│   │   ├── /payment-intent
│   │   │   └── POST (Create - auth)
│   │   ├── /confirm-payment
│   │   │   └── POST (Confirm - auth)
│   │   ├── /cancel
│   │   │   └── POST (Cancel - auth)
│   │   └── (PATCH - Update status - admin)
│   └── /count
│       └── GET
│
├── /integrations/printful
│   ├── /sync
│   │   ├── POST (Start sync - auth/admin)
│   │   └── /:id
│   │       └── GET (Check status - auth/admin)
│   ├── /products
│   │   ├── GET (List - auth/admin)
│   │   ├── /:id
│   │   │   └── GET (Get - auth/admin)
│   │   └── /import
│   │       └── POST (Import - auth/admin)
│   └── /orders
│       ├── POST (Create - auth/admin)
│       ├── /:id
│       │   ├── GET (Get - auth/admin)
│       │   ├── /confirm
│       │   │   └── POST (Confirm - auth/admin)
│       │   └── DELETE (Cancel - auth/admin)
│
├── /cart
│   ├── POST (Create - public)
│   ├── GET  (List - optional auth)
│   ├── /:id
│   │   ├── GET (Get - optional auth)
│   │   ├── /items
│   │   │   └── POST (Add - optional auth)
│   │   ├── /items/:itemId
│   │   │   ├── PUT (Update - optional auth)
│   │   │   └── DELETE (Remove - optional auth)
│   │   └── /checkout
│   │       └── POST (Checkout - optional auth)
│
├── /customizations
│   ├── /images
│   │   ├── GET (List - auth)
│   │   ├── POST (Upload - auth)
│   │   ├── /:id
│   │   │   ├── GET (Get - auth)
│   │   │   ├── DELETE (Delete - auth)
│   │   │   ├── /process
│   │   │   │   └── POST (Process - auth)
│   │   │   └── /link
│   │   │       └── POST (Link - auth)
│   │   └── /count
│   │       └── GET
│
└── /customers
    ├── /signin
    │   └── POST (Sign in - public)
    └── /signup
        └── POST (Sign up - public)
```

---

## Error Handling Flow

```
Request → Handler
    │
    ├─► Input Validation
    │   └─ Error? → 400 Bad Request
    │
    ├─► Authentication
    │   └─ Error? → 401 Unauthorized
    │
    ├─► Authorization
    │   └─ Error? → 403 Forbidden
    │
    ├─► Service Operation
    │   ├─ Printful API Error? → 502 Bad Gateway
    │   ├─ DynamoDB Error? → 500 Internal Server Error
    │   ├─ Not Found? → 404 Not Found
    │   └─ Success? → 200/201/202/204
    │
    └─► Response
        {
          "error": {
            "code": "ERROR_CODE",
            "message": "Human readable message",
            "details": "Optional details"
          }
        }
```

---

## Security Layers

```
Request
  │
  ├─► HTTPS (TLS 1.3)
  │
  ├─► API Gateway Rate Limiting
  │   └─ 100 req/min (public)
  │   └─ 1000 req/min (authenticated)
  │   └─ 5000 req/min (admin)
  │
  ├─► CORS Middleware
  │   └─ Check origin headers
  │
  ├─► Authentication Middleware
  │   └─ JWT Bearer token validation
  │
  ├─► Authorization Middleware (TODO)
  │   └─ Admin role check for Printful endpoints
  │
  ├─► Input Validation
  │   └─ Type checking, range validation
  │
  └─► Response
      └─ No sensitive data in logs
      └─ Errors don't leak system info
```

