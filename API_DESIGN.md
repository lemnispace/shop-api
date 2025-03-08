# LemniSpace E-Commerce API Documentation

## Overview

This documentation covers the REST API endpoints for the LemniSpace e-commerce platform. The API provides functionality for managing products, collections, carts, orders, customizations, and integrations with fulfillment partners like Printful.

## Base URL

```bash
https://api.lemnispace.com/v1
```

## Authentication

All API requests require authentication using Bearer tokens.

```bash
Authorization: Bearer {your_access_token}
```

## API Endpoints

### Products

#### List Products

```bash
GET /products
```

Retrieves a paginated list of products.

**Query Parameters:**

- `limit` (integer, optional): Number of products to return (default: 20)
- `cursor` (string, optional): Pagination cursor
- `status` (string, optional): Filter by status (`active`, `draft`, `archived`)
- `collection` (string, optional): Filter by collection ID

**Response:** 200 OK

```json
{
  "products": [
    {
      "id": "prod_123456789",
      "title": "Custom Canvas Print",
      "description": "High-quality custom canvas printing",
      "status": "active",
      "price": 59.99,
      "sku": "CANVAS-001",
      "tags": ["canvas", "custom", "print"],
      "images": [
        {
          "id": "img_123456",
          "url": "https://cdn.lemnispace.com/images/canvas-001.jpg",
          "altText": "Canvas print example"
        }
      ],
      "variants": [
        {
          "id": "var_123456",
          "title": "12x16 Canvas",
          "price": 59.99,
          "sku": "CANVAS-001-12x16",
          "inventory": 100,
          "options": [
            {
              "name": "Size",
              "value": "12x16"
            }
          ],
          "dimensions": {
            "width": 12,
            "height": 16,
            "depth": 0.75,
            "weight": 1.2
          },
          "fulfillmentData": {
            "partnerId": "printful",
            "partnerProductId": "4564",
            "partnerVariantId": "4564-12x16"
          }
        }
      ],
      "createdAt": "2023-06-01T12:00:00Z",
      "updatedAt": "2023-06-15T14:30:00Z"
    }
  ],
  "pagination": {
    "nextCursor": "cursor_abcdef123456",
    "hasMore": true
  }
}
```

#### Get Product

```bash
GET /products/{productId}
```

Retrieves detailed information about a specific product.

**Response:** 200 OK

```json
{
  "id": "prod_123456789",
  "title": "Custom Canvas Print",
  "description": "High-quality custom canvas printing",
  "status": "active",
  "price": 59.99,
  "sku": "CANVAS-001",
  "inventory": 100,
  "tags": ["canvas", "custom", "print"],
  "customFields": {
    "printArea": "12x16 inches",
    "material": "Premium cotton canvas"
  },
  "images": [
    {
      "id": "img_123456",
      "url": "https://cdn.lemnispace.com/images/canvas-001.jpg",
      "altText": "Canvas print example"
    }
  ],
  "variants": [
    {
      "id": "var_123456",
      "title": "12x16 Canvas",
      "price": 59.99,
      "sku": "CANVAS-001-12x16",
      "inventory": 100,
      "options": [
        {
          "name": "Size",
          "value": "12x16"
        }
      ],
      "dimensions": {
        "width": 12,
        "height": 16,
        "depth": 0.75,
        "weight": 1.2
      },
      "fulfillmentData": {
        "partnerId": "printful",
        "partnerProductId": "4564",
        "partnerVariantId": "4564-12x16"
      }
    }
  ],
  "createdAt": "2023-06-01T12:00:00Z",
  "updatedAt": "2023-06-15T14:30:00Z"
}
```

#### Create Product

```bash
POST /products
```

Creates a new product.

**Request Body:**

```json
{
  "title": "Custom Canvas Print",
  "description": "High-quality custom canvas printing",
  "price": 59.99,
  "sku": "CANVAS-001",
  "status": "draft",
  "tags": ["canvas", "custom", "print"],
  "customFields": {
    "printArea": "12x16 inches",
    "material": "Premium cotton canvas"
  },
  "variants": [
    {
      "title": "12x16 Canvas",
      "price": 59.99,
      "sku": "CANVAS-001-12x16",
      "inventory": 100,
      "options": [
        {
          "name": "Size",
          "value": "12x16"
        }
      ],
      "dimensions": {
        "width": 12,
        "height": 16,
        "depth": 0.75,
        "weight": 1.2
      },
      "fulfillmentData": {
        "partnerId": "printful",
        "partnerProductId": "4564",
        "partnerVariantId": "4564-12x16"
      }
    }
  ]
}
```

**Response:** 201 Created

```json
{
  "id": "prod_123456789",
  "title": "Custom Canvas Print",
  "description": "High-quality custom canvas printing",
  "status": "draft",
  "price": 59.99,
  "sku": "CANVAS-001",
  "tags": ["canvas", "custom", "print"],
  "customFields": {
    "printArea": "12x16 inches",
    "material": "Premium cotton canvas"
  },
  "variants": [
    {
      "id": "var_123456",
      "title": "12x16 Canvas",
      "price": 59.99,
      "sku": "CANVAS-001-12x16",
      "inventory": 100,
      "options": [
        {
          "name": "Size",
          "value": "12x16"
        }
      ],
      "dimensions": {
        "width": 12,
        "height": 16,
        "depth": 0.75,
        "weight": 1.2
      },
      "fulfillmentData": {
        "partnerId": "printful",
        "partnerProductId": "4564",
        "partnerVariantId": "4564-12x16"
      }
    }
  ],
  "createdAt": "2023-06-01T12:00:00Z",
  "updatedAt": "2023-06-01T12:00:00Z"
}
```

#### Update Product

```bash
PUT /products/{productId}
```

Updates an existing product.

**Request Body:** (Same format as Create Product)

**Response:** 200 OK (Same format as Get Product)

#### Delete Product

```bash
DELETE /products/{productId}
```

Deletes a product.

**Response:** 204 No Content

### Product Variants

#### Create Variant

```bash
POST /products/{productId}/variants
```

Adds a new variant to an existing product.

**Request Body:**

```json
{
  "title": "16x20 Canvas",
  "price": 79.99,
  "sku": "CANVAS-001-16x20",
  "inventory": 50,
  "options": [
    {
      "name": "Size",
      "value": "16x20"
    }
  ],
  "dimensions": {
    "width": 16,
    "height": 20,
    "depth": 0.75,
    "weight": 1.8
  },
  "fulfillmentData": {
    "partnerId": "printful",
    "partnerProductId": "4564",
    "partnerVariantId": "4564-16x20"
  }
}
```

**Response:** 201 Created

```json
{
  "id": "var_789012",
  "productId": "prod_123456789",
  "title": "16x20 Canvas",
  "price": 79.99,
  "sku": "CANVAS-001-16x20",
  "inventory": 50,
  "options": [
    {
      "name": "Size",
      "value": "16x20"
    }
  ],
  "dimensions": {
    "width": 16,
    "height": 20,
    "depth": 0.75,
    "weight": 1.8
  },
  "fulfillmentData": {
    "partnerId": "printful",
    "partnerProductId": "4564",
    "partnerVariantId": "4564-16x20"
  },
  "createdAt": "2023-06-15T14:30:00Z",
  "updatedAt": "2023-06-15T14:30:00Z"
}
```

#### Update Variant

```bash
PUT /products/{productId}/variants/{variantId}
```

Updates an existing product variant.

**Request Body:** (Same format as Create Variant)

**Response:** 200 OK (Same format as returned variant in Create Variant)

#### Delete Variant

```bash
DELETE /products/{productId}/variants/{variantId}
```

Deletes a product variant.

**Response:** 204 No Content

### Product Images

#### Upload Product Image

```bash
POST /products/{productId}/images
```

Uploads an image for a product.

**Request Body:** (multipart/form-data)

- `image`: The image file (supported formats: JPEG, PNG, WebP)
- `altText`: (optional) Alternative text for the image

**Response:** 201 Created

```json
{
  "id": "img_234567",
  "productId": "prod_123456789",
  "url": "https://cdn.lemnispace.com/images/canvas-002.jpg",
  "altText": "Canvas print side view",
  "width": 1200,
  "height": 800,
  "createdAt": "2023-06-15T15:30:00Z"
}
```

#### Associate Image with Variant

```bash
POST /products/{productId}/variants/{variantId}/images
```

Associates an existing product image with a specific variant.

**Request Body:**

```json
{
  "imageId": "img_234567"
}
```

**Response:** 200 OK

```json
{
  "success": true,
  "variantId": "var_123456",
  "imageId": "img_234567"
}
```

### Collections

#### List Collections

```bash
GET /collections
```

Retrieves all collections.

**Query Parameters:**

- `limit` (integer, optional): Number of collections to return (default: 20)
- `cursor` (string, optional): Pagination cursor

**Response:** 200 OK

```json
{
  "collections": [
    {
      "id": "col_123456",
      "title": "Home Decor",
      "description": "Beautiful customizable prints for your home",
      "productCount": 15,
      "createdAt": "2023-05-01T10:00:00Z",
      "updatedAt": "2023-06-01T12:00:00Z"
    }
  ],
  "pagination": {
    "nextCursor": "cursor_xyz789",
    "hasMore": false
  }
}
```

#### Get Collection

```bash
GET /collections/{collectionId}
```

Retrieves details for a specific collection, including its products.

**Query Parameters:**

- `includeProducts` (boolean, optional): Whether to include product details (default: true)
- `productLimit` (integer, optional): Number of products to return (default: 20)
- `productCursor` (string, optional): Pagination cursor for products

**Response:** 200 OK

```json
{
  "id": "col_123456",
  "title": "Home Decor",
  "description": "Beautiful customizable prints for your home",
  "products": [
    {
      "id": "prod_123456789",
      "title": "Custom Canvas Print",
      "description": "High-quality custom canvas printing",
      "status": "active",
      "price": 59.99,
      "images": [
        {
          "id": "img_123456",
          "url": "https://cdn.lemnispace.com/images/canvas-001.jpg",
          "altText": "Canvas print example"
        }
      ]
    }
  ],
  "productPagination": {
    "nextCursor": "cursor_prod123",
    "hasMore": true
  },
  "createdAt": "2023-05-01T10:00:00Z",
  "updatedAt": "2023-06-01T12:00:00Z"
}
```

#### Create Collection

```bash
POST /collections
```

Creates a new collection.

**Request Body:**

```json
{
  "title": "Home Decor",
  "description": "Beautiful customizable prints for your home",
  "productIds": ["prod_123456789", "prod_234567890"]
}
```

**Response:** 201 Created

```json
{
  "id": "col_123456",
  "title": "Home Decor",
  "description": "Beautiful customizable prints for your home",
  "productCount": 2,
  "createdAt": "2023-05-01T10:00:00Z",
  "updatedAt": "2023-05-01T10:00:00Z"
}
```

#### Update Collection

```bash
PUT /collections/{collectionId}
```

Updates an existing collection.

**Request Body:** (Same format as Create Collection)

**Response:** 200 OK (Same format as Get Collection)

#### Delete Collection

```bash
DELETE /collections/{collectionId}
```

Deletes a collection. This doesn't delete the products in the collection.

**Response:** 204 No Content

#### Add Products to Collection

```bash
POST /collections/{collectionId}/products
```

Adds products to a collection.

**Request Body:**

```json
{
  "productIds": ["prod_345678901", "prod_456789012"]
}
```

**Response:** 200 OK

```json
{
  "success": true,
  "collectionId": "col_123456",
  "productCount": 4
}
```

#### Remove Products from Collection

```bash
DELETE /collections/{collectionId}/products
```

Removes products from a collection.

**Request Body:**

```json
{
  "productIds": ["prod_345678901"]
}
```

**Response:** 200 OK

```json
{
  "success": true,
  "collectionId": "col_123456",
  "productCount": 3
}
```

### Cart

#### Create Cart

```bash
POST /cart
```

Creates a new shopping cart.

**Request Body:**

```json
{
  "customerId": "cus_123456789" // Optional
}
```

**Response:** 201 Created

```json
{
  "id": "cart_123456789",
  "customerId": "cus_123456789",
  "items": [],
  "totalPrice": 0.00,
  "createdAt": "2023-06-01T12:00:00Z",
  "updatedAt": "2023-06-01T12:00:00Z",
  "expiresAt": "2023-06-02T12:00:00Z"
}
```

#### Get Cart

```bash
GET /cart/{cartId}
```

Retrieves cart details.

**Response:** 200 OK

```json
{
  "id": "cart_123456789",
  "customerId": "cus_123456789",
  "items": [
    {
      "id": "item_123456",
      "productId": "prod_123456789",
      "variantId": "var_123456",
      "quantity": 2,
      "price": 59.99,
      "customizationData": {
        "imageUrl": "https://cdn.lemnispace.com/customizations/user-upload-123.jpg"
      },
      "product": {
        "title": "Custom Canvas Print",
        "image": "https://cdn.lemnispace.com/images/canvas-001.jpg"
      },
      "variant": {
        "title": "12x16 Canvas"
      }
    }
  ],
  "subtotal": 119.98,
  "estimatedTax": 10.80,
  "estimatedShipping": 5.99,
  "totalPrice": 136.77,
  "createdAt": "2023-06-01T12:00:00Z",
  "updatedAt": "2023-06-01T14:30:00Z",
  "expiresAt": "2023-06-02T12:00:00Z"
}
```

#### Add Item to Cart

```bash
POST /cart/{cartId}/items
```

Adds an item to the cart.

**Request Body:**

```json
{
  "productId": "prod_123456789",
  "variantId": "var_123456",
  "quantity": 2,
  "customizationData": {
    "imageId": "img_user_upload_123"
  }
}
```

**Response:** 200 OK

```json
{
  "id": "item_123456",
  "cartId": "cart_123456789",
  "productId": "prod_123456789",
  "variantId": "var_123456",
  "quantity": 2,
  "price": 59.99,
  "customizationData": {
    "imageUrl": "https://cdn.lemnispace.com/customizations/user-upload-123.jpg"
  },
  "createdAt": "2023-06-01T14:30:00Z"
}
```

#### Update Cart Item

```bash
PUT /cart/{cartId}/items/{itemId}
```

Updates a cart item, typically used to change quantity.

**Request Body:**

```json
{
  "quantity": 3
}
```

**Response:** 200 OK

```json
{
  "id": "item_123456",
  "cartId": "cart_123456789",
  "productId": "prod_123456789",
  "variantId": "var_123456",
  "quantity": 3,
  "price": 59.99,
  "customizationData": {
    "imageUrl": "https://cdn.lemnispace.com/customizations/user-upload-123.jpg"
  },
  "updatedAt": "2023-06-01T15:00:00Z"
}
```

#### Remove Item from Cart

```bash
DELETE /cart/{cartId}/items/{itemId}
```

Removes an item from the cart.

**Response:** 204 No Content

#### Get Cart Checkout URL

```bash
POST /cart/{cartId}/checkout
```

Generates a checkout URL for the cart.

**Response:** 200 OK

```json
{
  "checkoutUrl": "https://checkout.lemnispace.com/c/cart_123456789"
}
```

### Customizations

#### Upload Customization Image

```bash
POST /customizations/images
```

Uploads a customer image for product customization.

**Request Body:** (multipart/form-data)

- `image`: The image file (supported formats: JPEG, PNG, WebP)
- `cartId` (optional): Associate with a specific cart
- `productId` (optional): Associate with a specific product
- `variantId` (optional): Associate with a specific variant

**Response:** 201 Created

```json
{
  "id": "img_user_upload_123",
  "url": "https://cdn.lemnispace.com/customizations/user-upload-123.jpg",
  "width": 1200,
  "height": 1200,
  "createdAt": "2023-06-01T14:15:00Z",
  "expiresAt": "2023-06-08T14:15:00Z"
}
```

#### Process Customization Image

```bash
POST /customizations/images/{imageId}/process
```

Processes an uploaded image for customization (background removal, resize, crop, etc.).

**Request Body:**

```json
{
  "operations": [
    {
      "type": "removeBackground"
    },
    {
      "type": "resize",
      "width": 1000,
      "height": 1000,
      "maintainAspectRatio": true
    },
    {
      "type": "crop",
      "x": 100,
      "y": 100,
      "width": 800,
      "height": 800
    }
  ]
}
```

**Response:** 200 OK

```json
{
  "id": "img_user_upload_123_processed",
  "originalImageId": "img_user_upload_123",
  "url": "https://cdn.lemnispace.com/customizations/user-upload-123-processed.png",
  "width": 800,
  "height": 800,
  "createdAt": "2023-06-01T14:20:00Z",
  "expiresAt": "2023-06-08T14:20:00Z"
}
```

### Orders

#### List Orders

```bash
GET /orders
```

Retrieves a list of orders.

**Query Parameters:**

- `limit` (integer, optional): Number of orders to return (default: 20)
- `cursor` (string, optional): Pagination cursor
- `status` (string, optional): Filter by status
- `customerId` (string, optional): Filter by customer

**Response:** 200 OK

```json
{
  "orders": [
    {
      "id": "ord_123456789",
      "customerId": "cus_123456789",
      "status": "paid",
      "subtotal": 119.98,
      "tax": 10.80,
      "shipping": 5.99,
      "totalPrice": 136.77,
      "itemCount": 2,
      "createdAt": "2023-06-01T16:00:00Z",
      "updatedAt": "2023-06-01T16:05:00Z"
    }
  ],
  "pagination": {
    "nextCursor": "cursor_ord123",
    "hasMore": false
  }
}
```

#### Get Order

```bash
GET /orders/{orderId}
```

Retrieves detailed information about a specific order.

**Response:** 200 OK

```json
{
  "id": "ord_123456789",
  "customerId": "cus_123456789",
  "items": [
    {
      "id": "item_123456",
      "productId": "prod_123456789",
      "variantId": "var_123456",
      "quantity": 3,
      "price": 59.99,
      "total": 179.97,
      "customizationData": {
        "imageUrl": "https://cdn.lemnispace.com/customizations/user-upload-123.jpg"
      },
      "fulfillmentStatus": "pending",
      "product": {
        "title": "Custom Canvas Print",
        "image": "https://cdn.lemnispace.com/images/canvas-001.jpg"
      },
      "variant": {
        "title": "12x16 Canvas"
      }
    }
  ],
  "subtotal": 179.97,
  "tax": 16.20,
  "shipping": 7.99,
  "totalPrice": 204.16,
  "status": "paid",
  "shippingAddress": {
    "firstName": "John",
    "lastName": "Doe",
    "address1": "123 Main St",
    "city": "Anytown",
    "province": "CA",
    "country": "US",
    "zip": "12345",
    "phone": "+12345678901"
  },
  "billingAddress": {
    "firstName": "John",
    "lastName": "Doe",
    "address1": "123 Main St",
    "city": "Anytown",
    "province": "CA",
    "country": "US",
    "zip": "12345",
    "phone": "+12345678901"
  },
  "shippingMethod": "standard",
  "paymentMethod": "credit_card",
  "fulfillments": [
    {
      "id": "ful_123456",
      "status": "pending",
      "trackingNumber": null,
      "trackingUrl": null,
      "createdAt": "2023-06-01T16:10:00Z"
    }
  ],
  "createdAt": "2023-06-01T16:00:00Z",
  "updatedAt": "2023-06-01T16:10:00Z"
}
```

#### Create Order from Cart

```bash
POST /orders
```

Creates a new order from an existing cart.

**Request Body:**

```json
{
  "cartId": "cart_123456789",
  "customerId": "cus_123456789",
  "shippingAddress": {
    "firstName": "John",
    "lastName": "Doe",
    "address1": "123 Main St",
    "city": "Anytown",
    "province": "CA",
    "country": "US",
    "zip": "12345",
    "phone": "+12345678901"
  },
  "billingAddress": {
    "firstName": "John",
    "lastName": "Doe",
    "address1": "123 Main St",
    "city": "Anytown",
    "province": "CA",
    "country": "US",
    "zip": "12345",
    "phone": "+12345678901"
  },
  "shippingMethod": "standard",
  "paymentMethod": "credit_card",
  "paymentDetails": {
    "transactionId": "pm_123456789",
    "provider": "stripe"
  }
}
```

**Response:** 201 Created

```json
{
  "id": "ord_123456789",
  "customerId": "cus_123456789",
  "status": "pending",
  "subtotal": 179.97,
  "tax": 16.20,
  "shipping": 7.99,
  "totalPrice": 204.16,
  "createdAt": "2023-06-01T16:00:00Z",
  "updatedAt": "2023-06-01T16:00:00Z"
}
```

#### Update Order Status

```bash
PATCH /orders/{orderId}
```

Updates the status of an order.

**Request Body:**

```json
{
  "status": "paid"
}
```

**Response:** 200 OK

```json
{
  "id": "ord_123456789",
  "status": "paid",
  "updatedAt": "2023-06-01T16:05:00Z"
}
```

#### Cancel Order

```bash
POST /orders/{orderId}/cancel
```

Cancels an order if it hasn't been fulfilled yet.

**Response:** 200 OK

```json
{
  "id": "ord_123456789",
  "status": "cancelled",
  "updatedAt": "2023-06-01T17:00:00Z"
}
```

### Fulfillment

#### Create Fulfillment

```bash
POST /orders/{orderId}/fulfillments
```

Creates a fulfillment for an order.

**Request Body:**

```json
{
  "items": [
    {
      "orderItemId": "item_123456",
      "quantity": 3
    }
  ],
  "partnerId": "printful",
  "notifyCustomer": true
}
```

**Response:** 201 Created

```json
{
  "id": "ful_123456",
  "orderId": "ord_123456789",
  "status": "pending",
  "items": [
    {
      "id": "ful_item_123456",
      "orderItemId": "item_123456",
      "quantity": 3
    }
  ],
  "partnerId": "printful",
  "partnerOrderId": "printful_123456",
  "createdAt": "2023-06-01T16:10:00Z",
  "updatedAt": "2023-06-01T16:10:00Z"
}
```

#### Update Fulfillment Status

```bash
PATCH /orders/{orderId}/fulfillments/{fulfillmentId}
```

Updates the status of a fulfillment.

**Request Body:**

```json
{
  "status": "shipped",
  "trackingNumber": "1Z9999999999999999",
  "trackingUrl": "https://www.ups.com/track?tracknum=1Z9999999999999999",
  "notifyCustomer": true
}
```

**Response:** 200 OK

```json
{
  "id": "ful_123456",
  "orderId": "ord_123456789",
  "status": "shipped",
  "trackingNumber": "1Z9999999999999999",
  "trackingUrl": "https://www.ups.com/track?tracknum=1Z9999999999999999",
  "updatedAt": "2023-06-02T10:00:00Z"
}
```

#### Get Fulfillment

```bash
GET /orders/{orderId}/fulfillments/{fulfillmentId}
```

Retrieves details of a specific fulfillment.

**Response:** 200 OK

```json
{
  "id": "ful_123456",
  "orderId": "ord_123456789",
  "status": "shipped",
  "items": [
    {
      "id": "ful_item_123456",
      "orderItemId": "item_123456",
      "quantity": 3
    }
  ],
  "partnerId": "printful",
  "partnerOrderId": "printful_123456",
  "trackingNumber": "1Z9999999999999999",
  "trackingUrl": "https://www.ups.com/track?tracknum=1Z9999999999999999",
  "createdAt": "2023-06-01T16:10:00Z",
  "updatedAt": "2023-06-02T10:00:00Z"
}
```

### Fulfillment Partner Integration

#### Sync Products from Printful

```bash
POST /integrations/printful/sync
```

Syncs products from a Printful account.

**Response:** 202 Accepted

```json
{
  "syncId": "sync_123456",
  "status": "in_progress",
  "estimatedCompletion": "2023-06-01T17:00:00Z"
}
```

#### Get Sync Status

```bash
GET /integrations/sync/{syncId}
```

Gets the status of a sync operation.

**Response:** 200 OK

```json
{
  "id": "sync_123456",
  "type": "printful_products",
  "status": "completed",
  "progress": 100,
  "itemsProcessed": 150,
  "itemsTotal": 150,
  "startedAt": "2023-06-01T16:30:00Z",
  "completedAt": "2023-06-01T16:45:00Z"
}
```

#### Get Available Printful Products

```bash
GET /integrations/printful/products
```

Lists available products from Printful that can be imported.

**Query Parameters:**

- `limit` (integer, optional): Number of products to return (default: 20)
- `cursor` (string, optional): Pagination cursor
- `category` (string, optional): Filter by category

**Response:** 200 OK

```json
{
  "products": [
    {
      "id": "printful_4564",
      "title": "Canvas Print",
      "description": "High-quality custom canvas printing",
      "category": "Wall Art",
      "variants": [
        {
          "id": "printful_4564-12x16",
          "title": "12x16 Canvas",
          "price": 19.99,
          "dimensions": {
            "width": 12,
            "height": 16,
            "depth": 0.75,
            "weight": 1.2
          }
        },
        {
          "id": "printful_4564-16x20",
          "title": "16x20 Canvas",
          "price": 24.99,
          "dimensions": {
            "width": 16,
            "height": 20,
            "depth": 0.75,
            "weight": 1.8
          }
        }
      ],
      "mockupImageUrl": "https://files.cdn.printful.com/products/4564/mockup.jpg",
      "fileSpecs": {
        "format": "jpg,png",
        "minDpi": 150,
        "dimensions": {
          "width": 4800,
          "height": 6000
        }
      }
    }
  ],
  "pagination": {
    "nextCursor": "cursor_printful123",
    "hasMore": true
  }
}
```

#### Import Printful Product

```bash
POST /integrations/printful/products/import
```

Imports a Printful product into your store.

**Request Body:**

```json
{
  "printfulProductId": "printful_4564",
  "variantIds": ["printful_4564-12x16", "printful_4564-16x20"],
  "title": "Custom Canvas Print",
  "description": "High-quality custom canvas printing",
  "markupPercentage": 200
}
```

**Response:** 201 Created

```json
{
  "id": "prod_123456789",
  "title": "Custom Canvas Print",
  "description": "High-quality custom canvas printing",
  "status": "active",
  "variants": [
    {
      "id": "var_123456",
      "title": "12x16 Canvas",
      "price": 59.97,
      "fulfillmentData": {
        "partnerId": "printful",
        "partnerProductId": "4564",
        "partnerVariantId": "4564-12x16"
      }
    },
    {
      "id": "var_234567",
      "title": "16x20 Canvas",
      "price": 74.97,
      "fulfillmentData": {
        "partnerId": "printful",
        "partnerProductId": "4564",
        "partnerVariantId": "4564-16x20"
      }
    }
  ],
  "createdAt": "2023-06-01T17:00:00Z",
  "updatedAt": "2023-06-01T17:00:00Z"
}
```

## Errors

The API uses conventional HTTP status codes to indicate the success or failure of an API request:

- `200 OK`: The request was successful.
- `201 Created`: The resource was successfully created.
- `202 Accepted`: The request has been accepted for processing.
- `204 No Content`: The request was successful but there is no content to return.
- `400 Bad Request`: The request was invalid or cannot be served.
- `401 Unauthorized`: Authentication is required or has failed.
- `403 Forbidden`: The authenticated user doesn't have permission to access the requested resource.
- `404 Not Found`: The requested resource doesn't exist.
- `409 Conflict`: The request could not be completed due to a conflict with the current state of the resource.
- `422 Unprocessable Entity`: The request was well-formed but contains semantic errors.
- `429 Too Many Requests`: You've sent too many requests in a given amount of time.
- `500 Internal Server Error`: An error occurred on the server.

Error responses have the following format:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "The request was invalid",
    "details": [
      {
        "field": "title",
        "message": "Title is required"
      }
    ]
  }
}
```

## Rate Limiting

API calls are rate-limited to 100 requests per minute per API key. If you exceed this limit, you'll receive a `429 Too Many Requests` response.

Rate limit headers are included in API responses:

```bash
X-Rate-Limit-Limit: 100
X-Rate-Limit-Remaining: 90
X-Rate-Limit-Reset: 1623456789
```

## Webhooks

The API supports webhooks for event notifications. You can configure webhooks at `https://dashboard.lemnispace.com/settings/webhooks`.

Available events include:

- `product.created`
- `product.updated`
- `product.deleted`
- `order.created`
- `order.paid`
- `order.fulfilled`
- `order.cancelled`
- `fulfillment.updated`

Webhook requests will include a `X-Lemnispace-Signature` header for verification.
