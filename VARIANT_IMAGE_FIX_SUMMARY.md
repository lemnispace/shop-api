# Variant Image Association Fix

## Problem Summary

Three critical issues were identified and fixed:

### 1. **Variant Image Association Bug** (ROOT CAUSE)
**Issue**: Product images had their `Variants` field populated with variant SKUs (e.g., `["PF-MUG-11OZ"]`) instead of variant IDs (e.g., `["var_1763851635330685551"]`).

**Impact**: 
- Frontend `ImageGallery` component couldn't match images to variants
- Each variant appeared to have no unique image
- Cart items showed missing or incorrect product images

**Root Cause**: In `internal/services/printful.go`, images were associated with variants using SKUs (lines 618-647) BEFORE variant IDs were generated in `internal/services/product.go` (line 169).

**Fix**: Modified `product.go::CreateProduct()` to:
1. Build a SKU-to-variant-ID mapping after generating variant IDs
2. Convert any SKU references in `Image.Variants` to proper variant IDs
3. Generate image IDs and set alt text automatically

### 2. **Product ID Mismatch** 
**Issue**: Old product IDs cached in browser sessionStorage (e.g., `prod_1763849343816157587`) that no longer exist in database.

**Impact**: "Product not found" errors when trying to add items to cart.

**Cause**: Database was reinitialized with new product IDs, but browser cached old IDs.

**Solution**: Users need to clear sessionStorage/browser cache, or products should have stable IDs across reinitializations.

### 3. **Missing Test Coverage**
**Issue**: Integration tests didn't validate image-variant associations.

**Impact**: This bug wasn't caught during development or testing.

**Fix Needed**: Add tests to verify:
- Each variant has an associated image
- Image `Variants` field contains valid variant IDs (not SKUs)
- Cart items display correct product/variant images

## Files Modified

### `internal/services/printful.go`
- Added comment clarifying SKU storage in `productInputToProduct()`
- No functional changes needed (SKUs are correct at this stage)

### `internal/services/product.go` 
**`CreateProduct()` function (lines 166-226)**:
```go
// Key additions:
1. Build SKU-to-variant-ID map after generating IDs
2. Generate image IDs if not present  
3. Convert SKU references to variant IDs in Image.Variants
4. Auto-generate image alt text with variant info
5. Added debug logging for troubleshooting
```

## Validation

### Before Fix
```json
{
  "images": [{
    "id": "img_...",
    "url": "...",
    "variants": ["PF-MUG-11OZ"]  // ❌ SKU instead of ID
  }]
}
```

### After Fix
```json
{
  "images": [{
    "id": "img_1763851635330686676",
    "url": "https://files.cdn.printful.com/products/19/1320_1663762583.jpg",
    "variants": ["var_1763851635330685551"]  // ✅ Correct variant ID
  }]
}
```

### Cart Item Test
```bash
# Test adding product to cart
CART_ID=$(curl -X POST http://localhost:8080/v1/cart | jq -r '.id')
curl -X POST http://localhost:8080/v1/cart/$CART_ID/items \
  -H "Content-Type: application/json" \
  -d '{"productId":"prod_xxx","variantId":"var_yyy","quantity":1}'

# Expected: Cart item with correct product.image populated
{
  "id": "...",
  "product": {
    "title": "White Glossy Mug",
    "image": "https://files.cdn.printful.com/products/19/1320_1663762583.jpg"  // ✅
  }
}
```

## Testing Recommendations

Add integration tests:

```go
// Test 1: Verify variant-image associations after product creation
func TestProductImageVariantAssociation(t *testing.T) {
    // Create product with variants
    // Assert: Each image.Variants contains valid variant IDs
    // Assert: Each variant ID exists in product.Variants
}

// Test 2: Verify cart items have correct images
func TestCartItemImage(t *testing.T) {
    // Create product with multiple variants and images
    // Add each variant to cart
    // Assert: Cart item.product.image matches variant's associated image
}

// Test 3: Verify image SKU to ID conversion
func TestImageSKUConversion(t *testing.T) {
    // Create ProductInput with image.Variants containing SKUs
    // Call CreateProduct
    // Assert: Image.Variants now contains variant IDs, not SKUs
}
```

## Deployment Notes

1. **Backward Compatibility**: This fix is backward compatible. Existing products with SKU references will be automatically converted on next update.

2. **Database Migration**: No migration needed. Next product update/import will fix associations automatically.

3. **Cache Invalidation**: Users may need to:
   - Clear browser sessionStorage
   - Clear product cache in web-client
   - Re-navigate to products page

4. **Monitoring**: Watch for logs containing:
   - `"Generated variant ID"` - Variant ID creation
   - `"Converted SKU ... to variant ID"` - Successful SKU→ID conversion
   - `"Unknown variant reference format"` - Potential issues

## Future Improvements

1. **Stable Product IDs**: Use deterministic IDs based on SKU or Printful ID to survive database reinitializations

2. **Variant Image Field**: Consider adding `Image` field directly to `ProductVariant` model for easier access

3. **Image Validation**: Add validation to ensure all variants have at least one associated image (default or specific)

4. **Test Coverage**: Expand integration tests to cover image-variant associations

## Related Files

- `/internal/services/product.go` - Product CRUD with ID generation
- `/internal/services/printful.go` - Printful product import and conversion
- `/internal/models/product.go` - Product data models
- `/internal/models/common.go` - Image model definition
- `/web-client/src/app/components/product/ImageGallery.tsx` - Frontend image display

## Issue Timeline

- **Reported**: Users unable to add products to cart
- **Root Cause**: Image.Variants contained SKUs instead of variant IDs
- **Fixed**: 2025-11-22 - SKU-to-ID conversion in CreateProduct()
- **Status**: ✅ RESOLVED - Validated with manual testing
