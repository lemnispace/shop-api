#!/bin/bash
# Script to populate local DynamoDB with Printful product data
# This script creates an admin user and syncs the Printful catalog

set -e

echo "=== Populate Local DB with Printful Products ==="
echo ""

# Configuration
API_BASE_URL="${API_BASE_URL:-http://localhost:8080/v1}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@lemnispace.local}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123!@#}"
MAX_RETRIES=30
RETRY_DELAY=2

# Check if PRINTFUL_API_KEY is set
if [ -z "$PRINTFUL_API_KEY" ]; then
    echo "ERROR: PRINTFUL_API_KEY environment variable is not set"
    echo "Please set your Printful API key:"
    echo "  export PRINTFUL_API_KEY=your_api_key_here"
    echo ""
    echo "Get your API key from: https://www.printful.com/dashboard/settings"
    exit 1
fi

echo "✓ PRINTFUL_API_KEY is set"
echo "✓ Admin email: $ADMIN_EMAIL"
echo ""

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

    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE_URL/customers/register" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"$ADMIN_EMAIL\",
            \"password\": \"$ADMIN_PASSWORD\",
            \"firstName\": \"Admin\",
            \"lastName\": \"User\"
        }")

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

    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE_URL/customers/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"$ADMIN_EMAIL\",
            \"password\": \"$ADMIN_PASSWORD\"
        }")

    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | sed '$d')

    if [ "$HTTP_CODE" = "200" ]; then
        # Extract access token from JSON response
        ACCESS_TOKEN=$(echo "$BODY" | grep -o '"accessToken":"[^"]*' | cut -d'"' -f4)

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

# Function to sync Printful catalog
sync_printful_catalog() {
    local TOKEN=$1
    echo ""
    echo "Step 3: Syncing Printful catalog..."
    echo "This may take a few minutes depending on catalog size..."
    echo ""

    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE_URL/integrations/printful/sync" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json")

    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | sed '$d')

    if [ "$HTTP_CODE" = "202" ] || [ "$HTTP_CODE" = "200" ]; then
        echo "✓ Printful catalog sync started successfully"
        echo "Response: $BODY"
        echo ""
        echo "Note: Sync is running in the background. Check shop-api logs for progress."
        echo "You can monitor the sync status in the docker logs:"
        echo "  docker compose logs -f shop-api"
        return 0
    elif [ "$HTTP_CODE" = "403" ]; then
        echo "✗ Admin access denied (HTTP $HTTP_CODE)"
        echo "Make sure ADMIN_EMAILS environment variable includes: $ADMIN_EMAIL"
        echo "Current ADMIN_EMAILS: ${ADMIN_EMAILS:-<not set>}"
        echo ""
        echo "To fix this, add the following to your .env file:"
        echo "  ADMIN_EMAILS=$ADMIN_EMAIL"
        echo ""
        echo "Then restart docker compose:"
        echo "  docker compose down && docker compose up -d"
        exit 1
    else
        echo "✗ Failed to sync catalog (HTTP $HTTP_CODE)"
        echo "Response: $BODY"
        exit 1
    fi
}

# Function to import specific products for testing
import_sample_products() {
    local TOKEN=$1
    echo ""
    echo "Step 4: Importing sample products for testing..."
    echo ""

    # Common Printful product IDs for testing
    # 71: Unisex Staple T-Shirt
    # 19: Mug
    # 1: Poster
    PRODUCT_IDS=(71 19 1)

    for PRODUCT_ID in "${PRODUCT_IDS[@]}"; do
        echo "Importing product ID $PRODUCT_ID..."

        HTTP_RESPONSE=$(curl -s -w "HTTPCODE:%{http_code}" -X POST "$API_BASE_URL/integrations/printful/products/import" \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: application/json" \
            -d "{
                \"printfulProductId\": \"$PRODUCT_ID\",
                \"markupPercentage\": 30
            }")

        # Split HTTP response code from body
        HTTP_CODE=$(echo "$HTTP_RESPONSE" | sed -e 's/.*HTTPCODE://')
        BODY=$(echo "$HTTP_RESPONSE" | sed -e 's/HTTPCODE:.*//')

        if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "200" ]; then
            # Try to extract product title from response
            PRODUCT_TITLE=$(echo "$BODY" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('title', 'Product'))" 2>/dev/null || echo "Product $PRODUCT_ID")
            echo "  ✓ Successfully imported: $PRODUCT_TITLE"
        else
            echo "  ✗ Failed to import product $PRODUCT_ID (HTTP $HTTP_CODE)"
            # Try to extract error message
            ERROR_MSG=$(echo "$BODY" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('error', {}).get('message', 'Unknown error'))" 2>/dev/null || echo "$BODY")
            echo "  Error: $ERROR_MSG"
        fi

        sleep 2
    done
}

# Main execution
main() {
    wait_for_api
    register_admin
    TOKEN=$(login_admin)

    # Ask user which sync method to use
    echo ""
    echo "Choose sync method:"
    echo "  1. Full catalog sync (imports all products - slower but complete)"
    echo "  2. Sample products only (imports 3 test products - faster)"
    echo ""
    read -p "Enter choice (1 or 2) [default: 2]: " CHOICE
    CHOICE=${CHOICE:-2}

    if [ "$CHOICE" = "1" ]; then
        sync_printful_catalog "$TOKEN"
    else
        import_sample_products "$TOKEN"
    fi

    echo ""
    echo "=== Database Population Complete ==="
    echo ""
    echo "Summary:"
    echo "  ✓ Admin user created/verified: $ADMIN_EMAIL"
    echo "  ✓ Authentication successful"
    echo "  ✓ Products imported to local database"
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
