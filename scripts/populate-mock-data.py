#!/usr/bin/env python3
"""
Script to populate local DynamoDB with mock products that have proper structure.
Includes images and Printful-like metadata for testing the full checkout flow.
"""

import requests
import json
import time
import sys

API_BASE_URL = "http://localhost:8080/v1"
ADMIN_EMAIL = "admin@lemnispace.local"
ADMIN_PASSWORD = "admin123!@#"

def wait_for_api(max_retries=30, retry_delay=2):
    """Wait for API to be ready."""
    print("Waiting for shop-api to be ready...")
    for i in range(1, max_retries + 1):
        try:
            response = requests.get(f"{API_BASE_URL}/../health", timeout=5)
            if response.status_code == 200:
                print("✓ Shop-api is ready")
                return True
        except:
            pass
        print(f"  Attempt {i}/{max_retries}: API not ready yet, waiting {retry_delay}s...")
        time.sleep(retry_delay)
    print(f"✗ API did not become ready after {max_retries} attempts")
    return False

def login_admin():
    """Login and get JWT token."""
    print("\nLogging in as admin...")
    response = requests.post(
        f"{API_BASE_URL}/customers/login",
        json={
            "email": ADMIN_EMAIL,
            "password": ADMIN_PASSWORD
        }
    )

    if response.status_code == 200:
        data = response.json()
        print("✓ Successfully logged in as admin")
        return data['accessToken']
    else:
        print(f"✗ Failed to login (HTTP {response.status_code})")
        print(f"Response: {response.text}")
        sys.exit(1)

def create_product(token, product_data, product_name):
    """Create a product."""
    print(f"Creating product: {product_name}...")

    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }

    response = requests.post(
        f"{API_BASE_URL}/products",
        headers=headers,
        json=product_data
    )

    if response.status_code in [200, 201]:
        print(f"  ✓ Successfully created: {product_name}")
        return response.json()
    else:
        print(f"  ✗ Failed to create product (HTTP {response.status_code})")
        print(f"  Response: {response.text}")
        return None

def main():
    print("=== Populate Local DB with Mock Products ===\n")

    if not wait_for_api():
        sys.exit(1)

    token = login_admin()

    print("\nStep: Creating mock products with proper data...\n")

    # Product 1: T-Shirt with multiple variants
    tshirt = {
        "title": "AI Art Premium T-Shirt",
        "description": "Premium quality t-shirt with your custom AI-generated design. 100% combed and ring-spun cotton, comfortable fit.",
        "price": 2499,
        "sku": "AI-TSHIRT-001",
        "status": "active",
        "inventory": 100,
        "tags": ["apparel", "t-shirt", "cotton", "ai-art"],
        "images": [
            {
                "url": "https://files.cdn.printful.com/products/71/4011_1752236284.jpg",
                "altText": "AI Art Premium T-Shirt - Black",
                "isDefault": True,
                "position": 0,
                "variants": ["PF-BLACK-S", "PF-BLACK-M", "PF-BLACK-L"]
            },
            {
                "url": "https://files.cdn.printful.com/products/71/4026_1752236278.jpg",
                "altText": "AI Art Premium T-Shirt - White",
                "isDefault": False,
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
                    "requiresShipping": True
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
                    "requiresShipping": True
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
                    "requiresShipping": True
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
                    "partnerVariantId": "4024",
                    "requiresShipping": True
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
                    "requiresShipping": True
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
                    "requiresShipping": True
                }
            }
        ],
        "fulfillmentData": {
            "partnerId": "printful",
            "partnerProductId": "71",
            "requiresShipping": True
        }
    }

    # Product 2: Ceramic Mug
    mug = {
        "title": "Custom AI Art Ceramic Mug",
        "description": "High-quality 11oz ceramic mug with your custom AI-generated design. Microwave and dishwasher safe.",
        "price": 1499,
        "sku": "AI-MUG-001",
        "status": "active",
        "inventory": 100,
        "tags": ["drinkware", "mug", "ceramic", "ai-art"],
        "images": [
            {
                "url": "https://placehold.co/600x600/ffffff/333333/png?text=11oz+Mug",
                "altText": "Custom AI Art Ceramic Mug - 11oz",
                "isDefault": True,
                "position": 0,
                "variants": ["PF-MUG-11OZ"]
            },
            {
                "url": "https://placehold.co/700x700/ffffff/333333/png?text=15oz+Mug",
                "altText": "Custom AI Art Ceramic Mug - 15oz",
                "isDefault": False,
                "position": 1,
                "variants": ["PF-MUG-15OZ"]
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
                    "requiresShipping": True
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
                    "requiresShipping": True
                }
            }
        ],
        "fulfillmentData": {
            "partnerId": "printful",
            "partnerProductId": "19",
            "requiresShipping": True
        }
    }

    # Product 3: Canvas Print
    canvas = {
        "title": "AI Generated Canvas Print",
        "description": "Gallery-quality canvas print with your custom AI-generated artwork. Ready to hang.",
        "price": 3999,
        "sku": "AI-CANVAS-001",
        "status": "active",
        "inventory": 50,
        "tags": ["canvas", "print", "wall-art", "ai-art"],
        "images": [
            {
                "url": "https://placehold.co/600x800/e8e8e8/333333/png?text=12x16+Canvas",
                "altText": "AI Generated Canvas Print - 12x16",
                "isDefault": True,
                "position": 0,
                "variants": ["PF-CANVAS-12X16"]
            },
            {
                "url": "https://placehold.co/800x1000/e8e8e8/333333/png?text=16x20+Canvas",
                "altText": "AI Generated Canvas Print - 16x20",
                "isDefault": False,
                "position": 1,
                "variants": ["PF-CANVAS-16X20"]
            },
            {
                "url": "https://placehold.co/900x1350/e8e8e8/333333/png?text=18x24+Canvas",
                "altText": "AI Generated Canvas Print - 18x24",
                "isDefault": False,
                "position": 2,
                "variants": ["PF-CANVAS-18X24"]
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
                    "requiresShipping": True
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
                    "requiresShipping": True
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
                    "requiresShipping": True
                }
            }
        ],
        "fulfillmentData": {
            "partnerId": "printful",
            "partnerProductId": "1",
            "requiresShipping": True
        }
    }

    # Create all products
    create_product(token, tshirt, "AI Art Premium T-Shirt")
    create_product(token, mug, "Custom AI Art Ceramic Mug")
    create_product(token, canvas, "AI Generated Canvas Print")

    print("\n=== Database Population Complete ===\n")
    print("Summary:")
    print(f"  ✓ Admin user: {ADMIN_EMAIL}")
    print("  ✓ 3 products created with proper images and metadata\n")
    print("Products created:")
    print("  1. AI Art Premium T-Shirt (6 variants with color/size options)")
    print("  2. Custom AI Art Ceramic Mug (2 size variants)")
    print("  3. AI Generated Canvas Print (3 size variants)\n")
    print("Next steps:")
    print("  1. Start the web-client: cd ../web-client && npm run dev")
    print("  2. Visit: http://localhost:3000")
    print("  3. View products at: http://localhost:3000/shop/products\n")

if __name__ == "__main__":
    main()
