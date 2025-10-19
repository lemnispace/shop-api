#!/bin/bash
# Test script to verify real Printful API communication

set -e

echo "=== Printful API Integration Test ==="
echo ""

# Check if PRINTFUL_API_KEY is set
if [ -z "$PRINTFUL_API_KEY" ]; then
    echo "ERROR: PRINTFUL_API_KEY environment variable is not set"
    echo "Please set your Printful API key:"
    echo "  export PRINTFUL_API_KEY=your_api_key_here"
    exit 1
fi

echo "✓ PRINTFUL_API_KEY is set"
echo ""

# Test 1: Call Printful API directly with curl to verify credentials
echo "Test 1: Verifying API credentials with direct curl call..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer $PRINTFUL_API_KEY" \
    https://api.printful.com/products)

if [ "$HTTP_CODE" = "200" ]; then
    echo "✓ Successfully authenticated with Printful API (HTTP 200)"
else
    echo "✗ Failed to authenticate with Printful API (HTTP $HTTP_CODE)"
    if [ "$HTTP_CODE" = "401" ]; then
        echo "  ERROR: Invalid API key"
    fi
    exit 1
fi
echo ""

# Test 2: Fetch real product catalog
echo "Test 2: Fetching real product catalog from Printful..."
RESPONSE=$(curl -s -H "Authorization: Bearer $PRINTFUL_API_KEY" \
    https://api.printful.com/products)

PRODUCT_COUNT=$(echo "$RESPONSE" | grep -o '"id":[0-9]*' | wc -l)
if [ "$PRODUCT_COUNT" -gt 0 ]; then
    echo "✓ Retrieved $PRODUCT_COUNT products from Printful catalog"
    echo "  Sample product IDs:"
    echo "$RESPONSE" | grep -o '"id":[0-9]*' | head -5 | sed 's/^/    /'
else
    echo "✗ No products found in response"
    exit 1
fi
echo ""

# Test 3: Run Go integration test that uses real API
echo "Test 3: Running Go integration test with real Printful client..."
cd "$(dirname "$0")/.."

# Create a temporary test file
cat > /tmp/test_printful_real.go <<'EOF'
package main

import (
    "context"
    "fmt"
    "os"
    "github.com/lemnispace/shop-api/internal/services"
)

func main() {
    apiKey := os.Getenv("PRINTFUL_API_KEY")
    if apiKey == "" {
        fmt.Println("ERROR: PRINTFUL_API_KEY not set")
        os.Exit(1)
    }

    // Create real Printful client (not mock)
    client := services.NewPrintfulClient(apiKey, nil)

    // Make real API call
    products, err := client.GetProducts(context.Background())
    if err != nil {
        fmt.Printf("ERROR: Failed to get products: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("✓ Successfully retrieved %d products from Printful API\n", len(products))
    if len(products) > 0 {
        fmt.Printf("  First product: ID=%d, Name=%s\n", products[0].ID, products[0].Name)
    }
}
EOF

docker compose exec -T shop-api sh -c "cd /app && go run /tmp/test_printful_real.go"
rm /tmp/test_printful_real.go

echo ""
echo "=== All Tests Passed ==="
echo ""
echo "Summary:"
echo "  ✓ Direct API authentication confirmed"
echo "  ✓ Real product catalog retrieved"
echo "  ✓ Go client successfully communicates with Printful API"
echo ""
echo "The Printful integration is using the REAL API at https://api.printful.com"
