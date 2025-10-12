# 🎯 LemniSpace Shop-API Completion Game Plan

**Created:** 2025-10-12
**Last Updated:** 2025-10-12
**Goal:** Complete shop-api and migrate web-client from Shopify to enable full order completion

---

## 🎉 Recent Progress (2025-10-12)

### Completed Tasks
- ✅ **Task 1.1: Collection Service Implementation**
  - Full DynamoDB-backed collection service
  - Comprehensive integration tests (10/10 passing)
  - Cursor-based pagination working correctly

- ✅ **Task 1.2: GetCustomerCarts Handler**
  - Replaced panic with proper implementation
  - Integrated with existing CartService
  - Support for includeExpired query parameter

### Infrastructure Improvements
- ✅ Enhanced development environment with DevContainer support
- ✅ Improved docker-compose configuration
- ✅ Added Makefile commands for easier development
- ✅ Updated README with DevContainer workflow
- ✅ Fixed S3Service interface compilation issues
- ✅ Fixed DynamoDB cursor encoding/decoding for proper pagination

### 🎯 Critical Path to First Order (6-8 days)

**Current State:**
```
✅ Browse Products → ✅ Add to Cart → ❌ DEAD END (no orders)
```

**After Critical Path:**
```
✅ Browse → ✅ Cart → ✅ Order → ✅ Payment → ✅ Printful → ✅ SHIPPED!
```

**Priority Order (Revised for Speed):**

1. **🔄 Orders Service (Task 1.6)** ⏱️ 1-2 days - IN PROGRESS
   - **Why first:** Everything depends on orders existing
   - **Unblocks:** Payment, Fulfillment, Customer tracking
   - **Value:** Creates the order entity that flows through the system

2. **📋 Payment Integration (Task 1.7)** ⏱️ 1 day - NEXT
   - **Why second:** Can create orders but can't collect payment
   - **Unblocks:** Revenue generation
   - **Value:** Stripe integration for payment processing

3. **📋 Printful Integration (Task 1.9)** ⏱️ 2-3 days - NEXT
   - **Why third:** Paid orders need physical fulfillment
   - **Unblocks:** Product manufacturing and shipping
   - **Value:** Orders actually get fulfilled

4. **📋 Authentication (Tasks 1.10-11)** ⏱️ 2 days - NEXT
   - **Why fourth:** Need customer accounts for order tracking
   - **Unblocks:** Order history, secure checkout
   - **Value:** Production-ready security

**After this (6-8 days): FIRST COMPLETE ORDER POSSIBLE! 🎉**

### ⏸️ Deferred Until After MVP (Polish Features)

These add UX value but don't block order completion:

- **Task 1.3: Product Image Upload** (Printful catalog has images)
- **Task 1.4: Image Processing** (Customers can upload pre-processed)
- **Task 1.5: Error Handling** (Incremental improvement)
- **Task 1.8: Fulfillment Service** (Printful API provides tracking)

---

## 🧠 Why This Order? (Rationale)

### Critical Path Strategy
We prioritize **enabling complete orders ASAP** over polishing individual features.

**Old Approach (Feature-First):**
```
Complete ALL features → Then test flow → Find it doesn't work → Debug
Timeline: 3-4 weeks to first order
```

**New Approach (Critical Path):**
```
Build order flow → Test end-to-end → Find issues early → Iterate
Timeline: 6-8 days to first order
```

### Why Defer Image Features?

**Task 1.3 (Product Image Upload):**
- **Who needs it:** Admins uploading custom products
- **Workaround:** Printful sync provides product images
- **Impact of waiting:** Zero (can still sell products)
- **Value:** 10% (nice admin feature)

**Task 1.4 (Image Processing):**
- **Who needs it:** Customers customizing products
- **Workaround:** Upload pre-processed images
- **Impact of waiting:** Minor UX inconvenience
- **Value:** 15% (nice customer feature)

**Task 1.6 (Orders):**
- **Who needs it:** Everyone
- **Workaround:** None (blocks all orders)
- **Impact of waiting:** 100% (no revenue)
- **Value:** 100% (enables business)

### Business Impact

**Focus on Orders First:**
- ✅ Enables revenue in 6-8 days
- ✅ Tests critical integration points early
- ✅ Identifies architectural issues sooner
- ✅ Delivers business value fastest

**Polish Features Later:**
- ⏸️ Can add incrementally
- ⏸️ Don't block revenue
- ⏸️ Can be done in parallel with other work
- ⏸️ Lower risk to timeline

---

## 📊 Current State Assessment

### Web-Client Dependencies (Currently Shopify-Based)
```
✅ Mosaic Generation API → Proxies to txt-mosaic service
❌ Custom Products API → Creates products in Shopify (needs migration)
❌ Cart API → Uses Shopify Storefront API (needs migration)
❌ Sync API → Syncs Printful → Shopify (needs migration)
```

### Shop-API Implementation Status
```
✅ Products API (90%) - Image upload stubbed
✅ Product Variants API (100%)
✅ Collections API (100%) - COMPLETED with comprehensive tests
✅ Cart API (100%) - COMPLETED including GetCustomerCarts
⚠️ Customizations API (70%) - Image processing stubbed
❌ Orders API (0%)
❌ Fulfillment API (0%)
❌ Customers/Auth API (0%)
❌ Discounts API (0%)
❌ Printful Integration (0%)
```

---

## 🛠️ Development Environment Setup

**IMPORTANT**: This project uses DevContainers for development.

### Quick Start with DevContainer

1. **Open in VS Code**:
   - Open the `shop-api` folder in VS Code
   - When prompted, select "Reopen in Container"
   - VS Code will build and start the devcontainer with Go 1.23

2. **Inside DevContainer - Run Tests**:
   ```bash
   make test        # Run integration tests (starts DynamoDB + MinIO automatically)
   make test-clean  # Clean up test services
   ```

3. **Inside DevContainer - Run API**:
   ```bash
   make run         # Run API server (requires services to be running)
   ```

4. **From Host or DevContainer - Manage Services**:
   ```bash
   make dev-up      # Start all services (DynamoDB, MinIO, API)
   make dev-down    # Stop all services
   make dev-logs    # Follow service logs
   ```

### Service URLs (when running locally)
- **API**: http://localhost:8080
- **DynamoDB**: http://localhost:8000
- **MinIO**: http://localhost:9000
- **MinIO Console**: http://localhost:9001 (admin/minioadmin)

### Direct Go Commands (inside devcontainer)
```bash
go test ./...                      # Run all tests
go test ./internal/handlers/...    # Run specific package tests
go run ./cmd/shop                  # Run API directly
go build -o shop-api ./cmd/shop    # Build binary
```

### Without DevContainer (alternative)
If you prefer not to use devcontainers, ensure you have:
- Go 1.23+
- Docker & Docker Compose
- AWS CLI
- Make

Then run `make dev-up` to start services and `make test` to run tests.

---

## 🎯 PHASE 1: Complete Shop-API Backend
**Duration:** 2-3 weeks
**Goal:** Fully functional API supporting complete order flow

### PHASE 1A: Complete Existing Features ⏱️ 2-3 days

#### ✅ Task 1.1: Implement Collection Service (DynamoDB) - COMPLETED
**File:** `shop-api/internal/services/collection_impl.go`
**Status:** ✅ COMPLETED (2025-10-12)
**Priority:** HIGH

**Completed:**
1. ✅ Created `CollectionServiceImpl` struct with full DynamoDB implementation
2. ✅ Implemented all interface methods:
   - `CreateCollection` - Stores in DynamoDB with generated ID
   - `GetCollection` - Fetches with products via joins
   - `ListCollections` - Lists with cursor-based pagination
   - `UpdateCollection` - Updates collection metadata
   - `DeleteCollection` - Removes collection and relationships
   - `AddProductToCollection` - Creates collection-product links
   - `RemoveProductFromCollection` - Deletes links
   - `CountCollections` - Counts with optional filters
3. ✅ Implemented DynamoDB patterns:
   - PK: `COLLECTION#{id}`, SK: `METADATA` for collection data
   - PK: `COLLECTION#{id}`, SK: `PRODUCT#{productId}` for relationships
   - GSI1: For querying and filtering
4. ✅ Fixed cursor encoding/decoding to properly handle AttributeValue types
5. ✅ Comprehensive integration tests (10/10 passing):
   - CRUD operations tested
   - Pagination with cursor handling tested
   - Product relationships tested
   - Error cases tested

**Test Results:** All 10 tests passing in ~20 seconds

---

#### ✅ Task 1.2: Fix GetCustomerCarts Endpoint - COMPLETED
**File:** `shop-api/internal/handlers/cart.go:84`
**Status:** ✅ COMPLETED (2025-10-12)
**Priority:** MEDIUM

**Completed:**
1. ✅ Removed panic statement
2. ✅ Implemented proper handler with query parameter parsing
3. ✅ Integrated with existing `cartService.GetCartsByCustomer()` method
4. ✅ Added support for `includeExpired` query parameter (default: false)
5. ✅ Returns list of carts with count in response
6. ✅ Proper error handling and validation

**Implementation Details:**
- Uses existing CartService.GetCartsByCustomer method
- Query params: `customer` (required), `includeExpired` (optional, boolean)
- Response format: `{"carts": [...], "count": n}`
- Uses GSI1: `CUSTOMER#{customerId}` for efficient querying

---

#### Task 1.3: Implement Product Image Upload
**File:** `shop-api/internal/handlers/products.go:424`
**Status:** Stubbed with `panic("not implemented")`
**Priority:** HIGH

**What to do:**
1. Accept multipart/form-data file upload
2. Validate image format (JPEG, PNG, WebP)
3. Validate image size (max 10MB)
4. Generate unique image ID
5. Upload to S3 bucket: `products/{productId}/{imageId}.{ext}`
6. Create DynamoDB record:
   - PK: `PRODUCT#{productId}`, SK: `IMAGE#{imageId}`
7. Return image URL and metadata

**Acceptance Criteria:**
- POST `/v1/products/{id}/images` uploads to S3
- Image metadata stored in DynamoDB
- Returns presigned URL or public URL
- Validates file size and format

---

#### Task 1.4: Implement Image Processing Operations
**File:** `shop-api/internal/services/customization.go:308`
**Status:** Stubbed, returns original image
**Priority:** MEDIUM-HIGH

**What to do:**
1. Integrate image processing library (recommend: `github.com/disintegration/imaging`)
2. Implement operations:
   - **Resize**: Resize image with aspect ratio preservation
   - **Crop**: Crop image to specified dimensions
   - **Background Removal**: Integrate with external service or library
3. Store processed image in S3: `customizations/{userId}/{imageId}/processed.{ext}`
4. Update DynamoDB with processed image metadata
5. Return processed image URL

**Libraries to consider:**
```go
github.com/disintegration/imaging  // Resize, crop
github.com/anthonynsimon/bild      // Image processing
// For background removal, call external API
```

**Acceptance Criteria:**
- POST `/v1/customizations/images/{id}/process` with operations JSON
- Resize operation works correctly
- Crop operation works correctly
- Background removal integrated (can be external API)
- Processed image stored in S3
- Original image preserved

---

#### Task 1.5: Add Error Handling & Validation
**Files:** All handlers and services
**Status:** Some validation exists, needs comprehensive review
**Priority:** MEDIUM

**What to do:**
1. Add input validation for all endpoints
2. Return consistent error responses:
   ```json
   {
     "error": {
       "code": "VALIDATION_ERROR",
       "message": "Invalid input",
       "details": {...}
     }
   }
   ```
3. Add validation for:
   - Required fields
   - Field length limits
   - Valid enum values
   - Price ranges (positive numbers)
   - Valid product/variant IDs
4. Implement proper error handling in services
5. Log errors with context

**Acceptance Criteria:**
- All endpoints validate input
- Consistent error response format
- Helpful error messages
- No panics for invalid input

---

### PHASE 1B: Implement Orders System ⏱️ 3-4 days

#### Task 1.6: Implement Orders Service & Handler
**Files:**
- `shop-api/internal/services/order.go` (new)
- `shop-api/internal/handlers/orders.go` (new)

**Status:** Not started
**Priority:** CRITICAL

**What to do:**

**1. Create Order Service (`order.go`):**
```go
type OrderService interface {
    CreateOrder(ctx context.Context, input *models.OrderInput) (*models.Order, error)
    GetOrder(ctx context.Context, orderID string) (*models.Order, error)
    GetOrders(ctx context.Context, filters OrderFilters, pagination *Pagination) (*OrderList, error)
    UpdateOrderStatus(ctx context.Context, orderID string, status models.OrderStatus) error
    CancelOrder(ctx context.Context, orderID string) error
}
```

**2. DynamoDB Schema:**
```
Order:
  PK: ORDER#{orderId}
  SK: METADATA
  GSI1PK: CUSTOMER#{customerId}
  GSI1SK: ORDER#{timestamp}
  GSI2PK: STATUS#{status}
  GSI2SK: ORDER#{orderId}

Order Items:
  PK: ORDER#{orderId}
  SK: ITEM#{itemId}
```

**3. Order Creation Flow:**
```
1. Validate cart exists and not empty
2. Lock inventory for items
3. Calculate final prices (subtotal, tax, shipping, discount)
4. Create order in DynamoDB
5. Copy cart items to order items
6. Update product inventory
7. Mark cart as checked out
8. Return order with checkout URL
```

**4. Implement Handlers:**
- `POST /v1/orders` - Create order from cart
- `GET /v1/orders/:id` - Get order details
- `GET /v1/orders` - List orders (with customer filter)
- `PATCH /v1/orders/:id` - Update order status
- `POST /v1/orders/:id/cancel` - Cancel order

**Acceptance Criteria:**
- Can create order from cart
- Order stores all cart items
- Inventory is updated
- Order has status tracking (pending, paid, fulfilled, shipped, delivered, cancelled)
- Customer can view their orders
- Orders are paginated
- Order includes pricing breakdown

---

#### Task 1.7: Implement Payment Integration (Stripe)
**Files:**
- `shop-api/internal/services/payment.go` (new)
- `shop-api/internal/handlers/payment.go` (new)

**Status:** Not started
**Priority:** CRITICAL

**What to do:**
1. Install Stripe Go SDK: `go get github.com/stripe/stripe-go/v78`
2. Create Payment Service:
   ```go
   type PaymentService interface {
       CreatePaymentIntent(amount int64, currency string, metadata map[string]string) (*stripe.PaymentIntent, error)
       ConfirmPayment(paymentIntentID string) error
       RefundPayment(paymentIntentID string, amount int64) error
   }
   ```
3. Add payment endpoints:
   - `POST /v1/orders/:id/payment-intent` - Create Stripe PaymentIntent
   - `POST /v1/orders/:id/confirm-payment` - Confirm payment
4. Update order status after successful payment
5. Handle webhooks from Stripe

**Environment Variables:**
```bash
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
```

**Acceptance Criteria:**
- Can create Stripe PaymentIntent for order
- Payment confirmation updates order status to "paid"
- Webhook handles payment events
- Failed payments handled gracefully

---

### PHASE 1C: Implement Fulfillment & Printful ⏱️ 4-5 days

#### Task 1.8: Implement Fulfillment Service
**Files:**
- `shop-api/internal/services/fulfillment.go` (new)
- `shop-api/internal/handlers/fulfillment.go` (new)

**Status:** Not started
**Priority:** CRITICAL

**What to do:**
1. Create Fulfillment Service:
   ```go
   type FulfillmentService interface {
       CreateFulfillment(ctx context.Context, input *models.FulfillmentInput) (*models.Fulfillment, error)
       GetFulfillment(ctx context.Context, fulfillmentID string) (*models.Fulfillment, error)
       UpdateFulfillmentStatus(ctx context.Context, fulfillmentID string, status models.FulfillmentStatus) error
       GetOrderFulfillments(ctx context.Context, orderID string) ([]*models.Fulfillment, error)
   }
   ```

2. DynamoDB Schema:
   ```
   PK: ORDER#{orderId}
   SK: FULFILLMENT#{fulfillmentId}
   ```

3. Add handlers:
   - `POST /v1/orders/:id/fulfillments` - Create fulfillment
   - `GET /v1/orders/:id/fulfillments/:fid` - Get fulfillment
   - `PATCH /v1/orders/:id/fulfillments/:fid` - Update status

**Acceptance Criteria:**
- Can create fulfillment for order
- Tracks fulfillment status
- Stores tracking number and URL
- Links to fulfillment partner (Printful)

---

#### Task 1.9: Implement Printful Integration
**Files:**
- `shop-api/internal/services/printful.go` (new)
- `shop-api/internal/handlers/integrations.go` (new)

**Status:** Not started
**Priority:** CRITICAL

**What to do:**

**1. Create Printful Service:**
```go
type PrintfulService interface {
    // Catalog
    GetProducts(ctx context.Context) ([]PrintfulProduct, error)
    GetProduct(ctx context.Context, productID int) (*PrintfulProduct, error)
    GetVariant(ctx context.Context, variantID int) (*PrintfulVariant, error)

    // Orders
    CreateOrder(ctx context.Context, order *PrintfulOrderRequest) (*PrintfulOrder, error)
    GetOrder(ctx context.Context, orderID string) (*PrintfulOrder, error)

    // Sync
    SyncCatalog(ctx context.Context) error
}
```

**2. Implement API Client:**
- Base URL: `https://api.printful.com`
- Authentication: Bearer token
- Endpoints needed:
  - `GET /products` - List catalog
  - `GET /products/{id}` - Get product details
  - `POST /orders` - Submit order
  - `GET /orders/{id}` - Get order status

**3. Sync Products:**
- Fetch Printful catalog
- Create/update products in shop-api
- Store fulfillment data in product variants:
  ```json
  {
    "partnerId": "printful",
    "partnerProductId": "4564",
    "partnerVariantId": "4564-12x16"
  }
  ```

**4. Add Endpoints:**
- `POST /v1/integrations/printful/sync` - Sync catalog
- `GET /v1/integrations/printful/products` - List products
- `POST /v1/integrations/printful/products/import` - Import specific product
- `POST /v1/integrations/printful/orders` - Submit order to Printful

**Environment Variables:**
```bash
PRINTFUL_API_KEY=your_api_key
```

**Acceptance Criteria:**
- Can fetch Printful catalog
- Products sync to shop-api database
- Can submit order to Printful
- Fulfillment data tracked
- Webhook integration for order updates

---

### PHASE 1D: Implement Authentication ⏱️ 2-3 days

#### Task 1.10: Implement Customer Service
**Files:**
- `shop-api/internal/services/customer.go` (new)
- `shop-api/internal/handlers/customers.go` (new)

**Status:** Not started
**Priority:** HIGH

**What to do:**
1. Create Customer Service:
   ```go
   type CustomerService interface {
       CreateCustomer(ctx context.Context, input *models.CustomerInput) (*models.Customer, error)
       GetCustomer(ctx context.Context, customerID string) (*models.Customer, error)
       GetCustomerByEmail(ctx context.Context, email string) (*models.Customer, error)
       UpdateCustomer(ctx context.Context, customerID string, input *models.CustomerInput) error
       DeleteCustomer(ctx context.Context, customerID string) error
   }
   ```

2. DynamoDB Schema:
   ```
   PK: CUSTOMER#{customerId}
   SK: METADATA
   GSI1PK: EMAIL#{email}
   GSI1SK: CUSTOMER#{customerId}
   ```

3. Add handlers:
   - `POST /v1/customers` - Create customer
   - `GET /v1/customers/:id` - Get customer
   - `PUT /v1/customers/:id` - Update customer
   - `DELETE /v1/customers/:id` - Delete customer

**Acceptance Criteria:**
- Can create customer with email
- Email is unique
- Customer data stored securely
- Can look up by email

---

#### Task 1.11: Implement JWT Authentication
**Files:**
- `shop-api/internal/services/auth.go` (new)
- `shop-api/internal/middleware/auth.go` (new)

**Status:** Not started
**Priority:** HIGH

**What to do:**
1. Install JWT library: `go get github.com/golang-jwt/jwt/v5`
2. Create Auth Service:
   ```go
   type AuthService interface {
       Login(ctx context.Context, email, password string) (*models.CustomerLoginResponse, error)
       Register(ctx context.Context, input *models.CustomerInput) (*models.CustomerLoginResponse, error)
       RefreshToken(ctx context.Context, refreshToken string) (*models.CustomerLoginResponse, error)
       ValidateToken(tokenString string) (*jwt.Token, error)
   }
   ```

3. Password hashing with bcrypt:
   ```go
   import "golang.org/x/crypto/bcrypt"
   ```

4. Create auth middleware:
   - Extract JWT from Authorization header
   - Validate token
   - Add customer info to context

5. Add endpoints:
   - `POST /v1/customers/register` - Register with email/password
   - `POST /v1/customers/login` - Login with email/password
   - `POST /v1/auth/refresh` - Refresh access token

6. Protect endpoints:
   - Orders (require customer authentication)
   - Cart (associate with customer)
   - Customizations (user-specific)

**Environment Variables:**
```bash
JWT_SECRET=your_secret_key
JWT_EXPIRATION=1h
```

**Acceptance Criteria:**
- Can register customer with password
- Can login and receive JWT token
- Token validation middleware works
- Protected endpoints require valid token
- Refresh token flow works

---

### PHASE 1E: Implement Discounts (Optional) ⏱️ 1-2 days

#### Task 1.12: Implement Discount Service
**Files:**
- `shop-api/internal/services/discount.go` (new)
- `shop-api/internal/handlers/discounts.go` (new)

**Status:** Not started
**Priority:** LOW (can defer)

**What to do:**
1. Create Discount Service
2. Implement discount code validation
3. Apply discounts to cart
4. Track discount usage

**Can defer to post-MVP**

---

## 🎯 PHASE 2: Migrate Web-Client from Shopify to Shop-API
**Duration:** 1-2 weeks
**Goal:** Web-client uses shop-api instead of Shopify

### PHASE 2A: Create API Adapter Layer ⏱️ 2-3 days

#### Task 2.1: Create Shop-API Client
**File:** `web-client/src/lib/shop-api/client.ts` (new)

**What to do:**
1. Create API client class:
   ```typescript
   class ShopAPIClient {
     constructor(baseURL: string, apiKey?: string)

     // Products
     getProducts(filters?: ProductFilters): Promise<ProductList>
     getProduct(id: string): Promise<Product>

     // Cart
     createCart(input: CartInput): Promise<Cart>
     getCart(cartId: string): Promise<Cart>
     addToCart(cartId: string, items: CartItemInput[]): Promise<Cart>
     updateCartItem(cartId: string, itemId: string, quantity: number): Promise<Cart>
     removeCartItem(cartId: string, itemId: string): Promise<Cart>

     // Orders
     createOrder(cartId: string, input: OrderInput): Promise<Order>
     getOrder(orderId: string): Promise<Order>

     // Customizations
     uploadCustomization(file: File, userId: string): Promise<CustomizationUpload>
     processImage(imageId: string, operations: ImageOperation[]): Promise<ProcessedImage>
   }
   ```

2. Handle authentication (JWT tokens)
3. Error handling and retries
4. Type definitions matching shop-api models

**Acceptance Criteria:**
- Client can communicate with shop-api
- Type-safe API calls
- Error handling

---

#### Task 2.2: Create Response Adapters
**File:** `web-client/src/lib/shop-api/adapters.ts` (new)

**What to do:**
1. Create adapter functions to convert shop-api responses to Shopify-compatible format:
   ```typescript
   // Convert shop-api Product to Shopify format
   function adaptProduct(apiProduct: ShopAPIProduct): ShopifyProduct

   // Convert shop-api Cart to Shopify format
   function adaptCart(apiCart: ShopAPICart): ShopifyCart

   // Convert shop-api Order to Shopify format
   function adaptOrder(apiOrder: ShopAPIOrder): ShopifyOrder
   ```

2. This allows minimal changes to existing web-client components

**Acceptance Criteria:**
- Adapters convert all necessary fields
- Existing components work with adapted data

---

### PHASE 2B: Update API Routes ⏱️ 3-4 days

#### Task 2.3: Update Cart API Route
**File:** `web-client/src/app/api/cart/route.ts`

**What to do:**
1. Replace Shopify Storefront API calls with shop-api client calls
2. Use adapters to maintain compatible response format
3. Update cookie management for cart ID
4. Handle errors from shop-api

**Before:**
```typescript
// Calls Shopify Storefront API
const cart = await shopifyClient.cart.create(...)
```

**After:**
```typescript
// Calls shop-api
const apiCart = await shopAPIClient.createCart(...)
const cart = adaptCart(apiCart) // Adapt to Shopify format
```

**Acceptance Criteria:**
- GET /api/cart returns cart data
- POST /api/cart creates cart
- PATCH /api/cart adds items
- All responses match expected format

---

#### Task 2.4: Update Cart Line API Route
**File:** `web-client/src/app/api/cart/line/route.ts`

**What to do:**
1. Replace Shopify calls with shop-api calls
2. Update item quantity via shop-api
3. Adapt responses

**Acceptance Criteria:**
- PATCH /api/cart/line updates item quantity
- Response format matches expectations

---

#### Task 2.5: Rewrite Custom Products API
**File:** `web-client/src/app/api/products/route.ts`

**What to do:**
1. Replace Shopify product duplication with shop-api customization flow:
   ```typescript
   // Old: Duplicate product in Shopify
   // New: Upload customization image to shop-api
   const upload = await shopAPIClient.uploadCustomization(file, userId)

   // Process image if needed
   await shopAPIClient.processImage(upload.imageId, operations)

   // Link to cart item when added to cart
   ```

2. Remove Shopify-specific metadata
3. Update response format

**Acceptance Criteria:**
- POST /api/products uploads customization
- Customization linked to product variant
- Returns necessary IDs for cart

---

#### Task 2.6: Update Sync API Route
**File:** `web-client/src/app/api/sync/route.ts`

**What to do:**
1. Replace Shopify/Printful sync with shop-api sync endpoint
2. Call `POST /v1/integrations/printful/sync`
3. Return success status

**Acceptance Criteria:**
- POST /api/sync triggers shop-api sync
- Returns success/failure status

---

### PHASE 2C: Update Environment Configuration ⏱️ 1 day

#### Task 2.7: Update Environment Variables
**Files:**
- `web-client/.env.local`
- `web-client/.env.example`

**What to do:**
1. Add shop-api configuration:
   ```bash
   SHOP_API_URL=http://localhost:8080/v1
   SHOP_API_KEY=your_api_key  # If using API key auth
   ```

2. Remove Shopify variables (or keep for gradual migration):
   ```bash
   # Can remove these after full migration
   # LEMNISPACE_PRODUCTS_API_TOKEN
   # LEMNISPACE_PRODUCTS_ADMIN_API_TOKEN
   # etc.
   ```

3. Keep txt-mosaic service URL:
   ```bash
   TEXT_MOSAIC_API_URL=http://localhost:3001
   ```

**Acceptance Criteria:**
- Environment variables configured
- Web-client can connect to shop-api
- No Shopify credentials needed

---

### PHASE 2D: Test End-to-End Flow ⏱️ 2-3 days

#### Task 2.8: Test Complete Order Flow
**What to do:**
1. Start all services locally:
   ```bash
   # Terminal 1: Start shop-api services (from host or devcontainer)
   cd shop-api && make dev-up

   # Terminal 2: Start txt-mosaic
   cd txt-mosaic && cd app && uvicorn main:app --reload

   # Terminal 3: Start web-client
   cd web-client && npm run dev
   ```

2. Test complete flow:
   - Browse products
   - View product details
   - Upload custom image
   - Process image (crop, resize)
   - Add to cart
   - Update cart quantity
   - Proceed to checkout
   - Create order
   - Complete payment
   - Check order status
   - Verify Printful submission

3. Fix any issues found

**Acceptance Criteria:**
- Can complete full order without errors
- Order appears in shop-api
- Printful receives order
- Customer receives order confirmation

---

## 🎯 PHASE 3: Polish & Deploy ⏱️ 1 week

### Task 3.1: Add Logging & Monitoring
- CloudWatch logs for shop-api
- Error tracking (Sentry)
- Performance monitoring

### Task 3.2: Add Tests
- Unit tests for services
- Integration tests for endpoints
- E2E tests for critical flows

### Task 3.3: Deploy to AWS
- Deploy shop-api Lambda
- Configure API Gateway
- Set up DynamoDB tables
- Configure S3 buckets
- Deploy web-client (Vercel/Netlify)

### Task 3.4: Documentation
- Update API documentation
- Create deployment guide
- Write troubleshooting guide

---

## 📋 Summary Timeline (REVISED)

| Phase | Duration | Priority | Status |
|-------|----------|----------|--------|
| **✅ Foundation** | 2 days | HIGH | COMPLETED |
| **🎯 Critical Path (Orders → Payment → Printful → Auth)** | 6-8 days | CRITICAL | IN PROGRESS |
| **🚀 Web-Client Migration** | 4-5 days | HIGH | NOT STARTED |
| **🎨 Polish Features** | 2-3 days | LOW | DEFERRED |
| **📦 Deploy & Monitor** | 2-3 days | MEDIUM | NOT STARTED |
| **TOTAL TO MVP** | **~3 weeks** | | |

### Detailed Breakdown

**Week 1 (Now):**
- ✅ Collections Service (done)
- ✅ GetCustomerCarts (done)
- 🔄 Orders Service (1-2 days)
- Payment Integration (1 day)
- Printful Integration (2-3 days)

**Week 2:**
- Authentication (2 days)
- Web-Client Migration (4-5 days)

**Week 3:**
- E2E Testing (2-3 days)
- Deploy & Monitor (2-3 days)
- **🎉 PRODUCTION LAUNCH**

**Post-Launch:**
- Image Upload
- Image Processing
- Error Handling
- Fulfillment Service

---

## 🚀 Quick Start - Where to Begin

**Recommended Order (REVISED - Critical Path First):**

### ✅ Foundation (COMPLETED)
1. ✅ **Task 1.1: Collection Service** - COMPLETED
2. ✅ **Task 1.2: GetCustomerCarts** - COMPLETED

### 🎯 Critical Path (Enables First Order - 6-8 days)
3. 🔄 **Task 1.6: Orders Service** - IN PROGRESS
4. 📋 **Task 1.7: Payment Integration** - NEXT
5. 📋 **Task 1.9: Printful Integration** - NEXT
6. 📋 **Task 1.10 & 1.11: Auth** - NEXT

**🎉 Milestone: First Complete Order Possible!**

### 🚀 Integration & Migration
7. 📋 **Task 2.1-2.6: Migrate Web-Client** (connects web-client to shop-api)
8. 📋 **Task 2.8: E2E Testing** (verify complete flow)

**🎉 Milestone: Production Ready!**

### 🎨 Polish (Post-MVP)
9. ⏸️ **Task 1.3: Image Upload** (deferred - nice to have)
10. ⏸️ **Task 1.4: Image Processing** (deferred - nice to have)
11. ⏸️ **Task 1.5: Error Handling** (incremental improvement)
12. ⏸️ **Task 1.8: Fulfillment Service** (deferred - Printful handles)

---

## 📝 Notes & Considerations

### Data Migration
- No Shopify data to migrate (fresh start)
- Will need to sync Printful catalog initially

### Breaking Changes
- Web-client API will change (but adapters minimize impact)
- Cookie structure may change
- Cart ID format will change

### Backward Compatibility
- Keep txt-mosaic service as-is (already working)
- Web-client API routes act as translation layer

### Future Enhancements (Post-MVP)
- Admin panel for managing products
- Customer dashboard
- Email notifications
- Webhooks for order updates
- Advanced discount features
- Inventory management
- Analytics and reporting

---

## ✅ Success Criteria

**Phase 1 Complete When:**
- [ ] All endpoints return 200 for valid requests
- [ ] Can create order from cart
- [ ] Order submitted to Printful
- [ ] Authentication works
- [ ] All tests pass

**Phase 2 Complete When:**
- [ ] Web-client successfully calls shop-api
- [ ] No more Shopify API calls
- [ ] All existing features work

**Project Complete When:**
- [ ] Can complete full order: Browse → Cart → Checkout → Order → Fulfillment
- [ ] Printful receives order
- [ ] Customer can track order
- [ ] Deployed to production
- [ ] Documentation complete

---

**Ready to start? Begin with Task 1.1: Implement Collection Service!** 🚀
