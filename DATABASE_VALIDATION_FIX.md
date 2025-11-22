# Database Validation Fix

**Date**: November 22, 2025  
**Issue**: Corrupt products in local database with missing required fields (variants, images)

## Problems Identified

1. **Missing Variants**: Some products had no variants
2. **Missing Images**: Products and variants lacked required images
3. **No Validation**: No validation was performed before saving products to DynamoDB

## Root Cause

The Printful product import service (`internal/services/printful.go`) was not validating products before saving them to the database. This allowed corrupt/incomplete data to be persisted.

## Solution Implemented

### 1. Added Product Validation Function

Created `validateProductData()` in `printful.go` that checks:

- ✅ Product has a title
- ✅ Product has a SKU
- ✅ Product has at least one variant
- ✅ Each variant has a SKU, title, and positive price
- ✅ Product has at least one image
- ⚠️ Warns if variants are missing specific images (but allows fallback to default)

### 2. Updated `convertPrintfulProduct()`

Added validation call after product conversion:

```go
// VALIDATION: Ensure product has required data
if err := validateProductData(product); err != nil {
    return nil, fmt.Errorf("product validation failed: %w", err)
}
```

### 3. Database Reset

- Cleared all data: `docker compose down -v`
- Restarted services: `docker compose up -d`
- Repopulated with validated data: `python3 scripts/populate-mock-data.py`

## Validation Rules

### Required Fields (Will Fail)

- Product must have a title
- Product must have a SKU
- Product must have at least one variant
- Each variant must have SKU, title, and positive price
- Product must have at least one image

### Optional Fields (Will Warn)

- Variants without specific images (will use default image)

## Testing

After implementing validation:

```bash
# 1. Clear database
docker compose down -v

# 2. Start services
docker compose up -d

# 3. Populate with validated data
python3 scripts/populate-mock-data.py

# 4. Verify data
curl http://localhost:8080/v1/products | jq '.products[] | {title, variants: .variants | length, images: .images | length}'
```

## Results

All 3 mock products created successfully with:
- ✅ Multiple variants per product
- ✅ Images associated with variants
- ✅ Complete fulfillment data
- ✅ Proper Printful integration metadata

## Future Improvements

1. Add validation for imported Printful products in catalog sync
2. Add database migration to validate existing products
3. Add product health check endpoint
4. Implement product validation in product service layer (not just Printful)

## Files Changed

- `shop-api/internal/services/printful.go` - Added validation function
- `web-client/next.config.js` - Added localhost:9000 to allowed image domains
- `web-client/src/app/components/product/ImageGallery.tsx` - Fixed duplicate keys
- `web-client/src/app/components/product/ProductView.tsx` - Fixed variant selection
- `web-client/src/app/components/product/ProductSelectionForm.tsx` - Fixed customization logic
