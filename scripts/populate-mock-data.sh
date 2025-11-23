#!/bin/bash
# Script to populate local DynamoDB with mock products that have proper structure
# Includes images and Printful-like metadata for testing the full checkout flow

set -e

echo "=== Populate Local DB with Mock Products ==="
echo ""

# Configuration
API_BASE_URL="${API_BASE_URL:-http://localhost:8080/v1}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@lemnispace.local}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123!@#}"
MAX_RETRIES=30
RETRY_DELAY=2

# Function to wait for API to be ready
wait_for_api() {
    echo "Waiting for shop-api to be ready..."
    for i in $(seq 1 $MAX_RETRIES); do
        if curl -s -f "$API_BASE_URL/../health" > /dev/null 2>&1; then
            echo "✓ Shop-api is ready"
            return 0
        fi
        echo "  Attempt $i/$MAX_RETRIES: API not ready yet, waiting ${RETRY_DELAY}s..."
        sleep $RETRY_DELAY
    done
    echo "✗ API did not become ready after $MAX_RETRIES attempts"
    exit 1
}

# Function to register admin user
register_admin() {
    echo ""
    echo "Step 1: Registering admin user..."

    cat > /tmp/register.json << 'EOF'
{
    "email": "admin@lemnispace.local",
    "password": "admin123!@#",
    "firstName": "Admin",
    "lastName": "User"
}
EOF

    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE_URL/customers/register" \
        -H "Content-Type: application/json" \
        -d @/tmp/register.json)

    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | sed '$d')

    if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "200" ]; then
        echo "✓ Admin user registered successfully"
        return 0
    elif echo "$BODY" | grep -q "already exists"; then
        echo "✓ Admin user already exists, skipping registration"
        return 0
    else
        echo "✗ Failed to register admin user (HTTP $HTTP_CODE)"
        echo "Response: $BODY"
        exit 1
    fi
}

# Function to login and get JWT token
login_admin() {
    echo ""
    echo "Step 2: Logging in as admin..."

    cat > /tmp/login.json << 'EOF'
{
    "email": "admin@lemnispace.local",
    "password": "admin123!@#"
}
EOF

    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE_URL/customers/login" \
        -H "Content-Type: application/json" \
        -d @/tmp/login.json)

    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | sed '$d')

    if [ "$HTTP_CODE" = "200" ]; then
        ACCESS_TOKEN=$(echo "$BODY" | python3 -c "import sys, json; print(json.load(sys.stdin)['accessToken'])")

        if [ -z "$ACCESS_TOKEN" ]; then
            echo "✗ Failed to extract access token from response"
            echo "Response: $BODY"
            exit 1
        fi

        echo "✓ Successfully logged in as admin"
        echo "$ACCESS_TOKEN"
        return 0
    else
        echo "✗ Failed to login (HTTP $HTTP_CODE)"
        echo "Response: $BODY"
        exit 1
    fi
}

# Function to create a product
create_product() {
    local TOKEN=$1
    local PRODUCT_JSON=$2
    local PRODUCT_NAME=$3

    echo "Creating product: $PRODUCT_NAME..."

    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE_URL/products" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d "$PRODUCT_JSON")

    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | sed '$d')

    if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "200" ]; then
        echo "  ✓ Successfully created: $PRODUCT_NAME"
    else
        echo "  ✗ Failed to create product (HTTP $HTTP_CODE)"
        echo "  Response: $BODY"
    fi
}

# Main execution
main() {
    wait_for_api
    register_admin
    TOKEN=$(login_admin)

    echo ""
    echo "Step 3: Creating mock products with proper data..."
    echo ""

    # Product 1: T-Shirt with multiple variants
    TSHIRT_JSON='
{
  "title": "AI Art Premium T-Shirt",
  "description": "Premium quality t-shirt with your custom AI-generated design. 100% cotton, comfortable fit.",
  "price": 2499,
  "sku": "AI-TSHIRT-001",
  "status": "active",
  "inventory": 100,
  "tags": ["apparel", "t-shirt", "cotton", "ai-art"],
  "images": [
    {
      "url": "https://files.cdn.printful.com/o/upload/product-catalog-img/a3/a310a19f34e62e4ebae3f81c1a28e21a_l",
      "altText": "AI Art Premium T-Shirt",
      "isDefault": true,
      "position": 0,
      "variants": ["PF-BLACK-S", "PF-BLACK-M", "PF-BLACK-L"]
    },
    {
      "url": "https://files.cdn.printful.com/o/upload/product-catalog-img/8d/8d3c77ba7156467fb2ebe16e5f2e81be_l",
      "altText": "AI Art Premium T-Shirt White",
      "isDefault": false,
      "position": 1,
      "variants": ["PF-WHITE-S", "PF-WHITE-M", "PF-WHITE-L"]
    }
  ],
  "variants": [
    {
      "sku": "PF-BLACK-S",
      "title": "Black / Small",
      "price": 2499,
      "inventory": 100,
      "options": [
        {"name": "Color", "value": "Black"},
        {"name": "Size", "value": "S"}
      ],
      "fulfillmentData": {
        "partnerId": "printful",
        "partnerProductId": "71",
        "partnerVariantId": "4012",
        "requiresShipping": true
      }
    },
    {
      "sku": "PF-BLACK-M",
      "title": "Black / Medium",
      "price": 2499,
      "inventory": 100,
      "options": [
        {"name": "Color", "value": "Black"},
        {"name": "Size", "value": "M"}
      ],
      "fulfillmentData": {
        "partnerId": "printful",
        "partnerProductId": "71",
        "partnerVariantId": "4013",
        "requiresShipping": true
      }
    },
    {
      "sku": "PF-BLACK-L",
      "title": "Black / Large",
      "price": 2499,
      "inventory": 100,
      "options": [
        {"name": "Color", "value": "Black"},
        {"name": "Size", "value": "L"}
      ],
      "fulfillmentData": {
        "partnerId": "printful",
        "partnerProductId": "71",
        "partnerVariantId": "4014",
        "requiresShipping": true
      }
    },
    {
      "sku": "PF-WHITE-S",
      "title": "White / Small",
      "price": 2499,
      "inventory": 100,
      "options": [
        {"name": "Color", "value": "White"},
        {"name": "Size", "value": "S"}
      ],
      "fulfillmentData": {
        "partnerId": "printful",
        "partnerProductId": "71",
        "partnerVariantID": "4024",
        "requiresShipping": true
      }
    },
    {
      "sku": "PF-WHITE-M",
      "title": "White / Medium",
      "price": 2499,
      "inventory": 100,
      "options": [
        {"name": "Color", "value": "White"},
        {"name": "Size", "value": "M"}
      ],
      "fulfillmentData": {
        "partnerId": "printful",
        "partnerProductId": "71",
        "partnerVariantId": "4025",
        "requiresShipping": true
      }
    },
    {
      "sku": "PF-WHITE-L",
      "title": "White / Large",
      "price": 2499,
      "inventory": 100,
      "options": [
        {"name": "Color", "value": "White"},
        {"name": "Size", "value": "L"}
      ],
      "fulfillmentData": {
        "partnerId": "printful",
        "partnerProductId": "71",
        "partnerVariantId": "4026",
        "requiresShipping": true
      }
    }
  ],
  "fulfillmentData": {
    "partnerId": "printful",
    "partnerProductId": "71",
    "requiresShipping": true
  }
}'

    create_product "$TOKEN" "$TSHIRT_JSON" "AI Art Premium T-Shirt"

    # Product 2: Ceramic Mug
    MUG_JSON='
{
  "title": "Custom AI Art Ceramic Mug",
  "description": "High-quality 11oz ceramic mug with your custom AI-generated design. Microwave and dishwasher safe.",
  "price": 1499,
  "sku": "AI-MUG-001",
  "status": "active",
  "inventory": 100,
  "tags": ["drinkware", "mug", "ceramic", "ai-art"],
  "images": [
    {
      "url": "https://files.cdn.printful.com/o/upload/product-catalog-img/2e/2e0aa9614f8dbb3a38f662f8bc531e2c_l",
      "altText": "Custom AI Art Ceramic Mug",
      "isDefault": true,
      "position": 0,
      "variants": ["PF-MUG-11OZ", "PF-MUG-15OZ"]
    }
  ],
  "variants": [
    {
      "sku": "PF-MUG-11OZ",
      "title": "11 oz",
      "price": 1499,
      "inventory": 100,
      "options": [
        {"name": "Size", "value": "11 oz"}
      ],
      "fulfillmentData": {
        "partnerId": "printful",
        "partnerProductId": "19",
        "partnerVariantId": "1318",
        "requiresShipping": true
      }
    },
    {
      "sku": "PF-MUG-15OZ",
      "title": "15 oz",
      "price": 1799,
      "inventory": 100,
      "options": [
        {"name": "Size", "value": "15 oz"}
      ],
      "fulfillmentData": {
        "partnerId": "printful",
        "partnerProductId": "19",
        "partnerVariantId": "1319",
        "requiresShipping": true
      }
    }
  ],
  "fulfillmentData": {
    "partnerId": "printful",
    "partnerProductId": "19",
    "requiresShipping": true
  }
}'

    create_product "$TOKEN" "$MUG_JSON" "Custom AI Art Ceramic Mug"

    # Product 3: Canvas Print
    CANVAS_JSON='
{
  "title": "AI Generated Canvas Print",
  "description": "Gallery-quality canvas print with your custom AI-generated artwork. Ready to hang.",
  "price": 3999,
  "sku": "AI-CANVAS-001",
  "status": "active",
  "inventory": 50,
  "tags": ["canvas", "print", "wall-art", "ai-art"],
  "images": [
    {
      "url": "https://files.cdn.printful.com/o/upload/product-catalog-img/1e/1e1fe67e2d3967d3e5a48d2f34f7a89f_l",
      "altText": "AI Generated Canvas Print",
      "isDefault": true,
      "position": 0,
      "variants": ["PF-CANVAS-12X16", "PF-CANVAS-16X20", "PF-CANVAS-18X24"]
    }
  ],
  "variants": [
    {
      "sku": "PF-CANVAS-12X16",
      "title": "12\" x 16\"",
      "price": 3999,
      "inventory": 50,
      "options": [
        {"name": "Size", "value": "12x16"}
      ],
      "fulfillmentData": {
        "partnerId": "printful",
        "partnerProductId": "1",
        "partnerVariantId": "4438",
        "requiresShipping": true
      }
    },
    {
      "sku": "PF-CANVAS-16X20",
      "title": "16\" x 20\"",
      "price": 4999,
      "inventory": 50,
      "options": [
        {"name": "Size", "value": "16x20"}
      ],
      "fulfillmentData": {
        "partnerId": "printful",
        "partnerProductId": "1",
        "partnerVariantId": "4439",
        "requiresShipping": true
      }
    },
    {
      "sku": "PF-CANVAS-18X24",
      "title": "18\" x 24\"",
      "price": 5999,
      "inventory": 50,
      "options": [
        {"name": "Size", "value": "18x24"}
      ],
      "fulfillmentData": {
        "partnerId": "printful",
        "partnerProductId": "1",
        "partnerVariantId": "4440",
        "requiresShipping": true
      }
    }
  ],
  "fulfillmentData": {
    "partnerId": "printful",
    "partnerProductId": "1",
    "requiresShipping": true
  }
}'

    create_product "$TOKEN" "$CANVAS_JSON" "AI Generated Canvas Print"

    echo ""
    echo "=== Database Population Complete ==="
    echo ""
    echo "Summary:"
    echo "  ✓ Admin user created/verified: $ADMIN_EMAIL"
    echo "  ✓ Authentication successful"
    echo "  ✓ 3 products created with proper images and metadata"
    echo ""
    echo "Products created:"
    echo "  1. AI Art Premium T-Shirt (6 variants with color/size options)"
    echo "  2. Custom AI Art Ceramic Mug (2 size variants)"
    echo "  3. AI Generated Canvas Print (3 size variants)"
    echo ""
    echo "Next steps:"
    echo "  1. Start the web-client: cd ../web-client && npm run dev"
    echo "  2. Visit: http://localhost:3000"
    echo "  3. View products at: http://localhost:3000/products"
    echo ""
    echo "API endpoints:"
    echo "  - Products: $API_BASE_URL/products"
    echo "  - Health: $API_BASE_URL/../health"
    echo ""
}

main
