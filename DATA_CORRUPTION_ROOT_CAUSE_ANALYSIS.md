# Data Corruption - Root Cause Analysis

**Date**: November 22, 2025  
**Incident**: Corrupt products persisting in database after supposedly clearing all data

## Executive Summary

Old corrupt product data (from November 15th) persisted in the database despite running `docker compose down -v`. Root cause: Docker Compose was using **bind mounts** instead of named volumes, so data directories on the host filesystem were not removed.

---

## Timeline

1. **November 15, 2025**: Products created without validation → corrupt data stored
2. **November 22, 2025 @ 21:21**: Cleared database with `docker compose down -v`
3. **November 22, 2025 @ 21:21**: Repopulated with validated mock data
4. **November 22, 2025 @ 21:27**: User reports corrupt product `prod_1763239966434418382` still exists
5. **November 22, 2025 @ 21:28**: Investigation reveals old data from November 15th
6. **November 22, 2025 @ 21:29**: Root cause identified - bind mounts not removed

---

## Root Cause

### Problem

The `docker-compose.yml` uses **bind mounts** to local directories:

```yaml
volumes:
  - ./dynamodb-data:/home/dynamodblocal/data  # Line 29
  - ./minio-data:/data                        # Line 44
  - ./localstack-data:/var/lib/localstack     # Line 16
```

### Why This Caused Issues

1. **Bind mounts vs Named volumes**:
   - `docker compose down -v` only removes **named volumes**
   - Bind-mounted directories on the host filesystem are **NOT removed**
   - Old data persisted in `./dynamodb-data/` from November 15th

2. **No cleanup documentation**:
   - No clear instructions on how to properly clear local data
   - Developers assumed `-v` flag would clear everything

3. **Not in .gitignore**:
   - Data directories could accidentally be committed
   - No protection against stale data

---

## Evidence

### Old Corrupt Product Data

```json
{
  "id": "prod_1763239966434418382",
  "title": "AI Art Premium Poster",
  "images": null,  // ❌ NO IMAGES
  "variants": [...], // Has variants but no images
  "createdAt": "2025-11-15T20:52:46.434419216Z"  // ❌ Nov 15th
}
```

### Directory Inspection

```bash
$ ls -la | grep -E "dynamodb-data|minio-data|localstack-data"
drwxr-xr-x   3  santiagogomez  staff   96 Nov 22 13:21 dynamodb-data
drwxr-xr-x   6  santiagogomez  staff  192 Nov 15 12:52 localstack-data  # ❌ Nov 15th!
drwxr-xr-x   5  santiagogomez  staff  160 Nov 21 21:28 minio-data
```

---

## Contributing Factors

### 1. No Product Validation (Fixed)

Products were saved without validation:
- Missing images
- Missing variants
- No SKU validation

**Fix**: Added `validateProductData()` function in `printful.go`

### 2. Misleading Docker Command

The `-v` flag suggests "volumes will be removed" but doesn't apply to bind mounts:

```bash
docker compose down -v  # ❌ Doesn't remove bind-mounted directories
```

### 3. No Data Cleanup Script

No documented procedure for clearing local development data.

---

## Solution Implemented

### 1. Remove Corrupt Data

```bash
# Stop containers
docker compose down

# Remove bind-mounted data directories
rm -rf dynamodb-data minio-data localstack-data

# Restart with clean state
docker compose up -d

# Populate with validated data
python3 scripts/populate-mock-data.py
```

### 2. Add to .gitignore

```gitignore
# Local Docker data directories
dynamodb-data/
minio-data/
localstack-data/
```

### 3. Add Product Validation

Added validation in `internal/services/printful.go`:

```go
func validateProductData(product *models.ProductInput) error {
    // Check title, SKU, variants, images
    if len(product.Images) == 0 {
        return fmt.Errorf("product must have at least one image")
    }
    // ... more validation
}
```

### 4. Document Cleanup Procedure

Created proper cleanup commands in project documentation.

---

## Prevention Measures

### Immediate Actions ✅

1. ✅ Removed all corrupt data from local directories
2. ✅ Added validation to prevent corrupt data
3. ✅ Updated .gitignore to exclude data directories
4. ✅ Documented proper cleanup procedure

### Recommended Improvements

1. **Use Named Volumes Instead of Bind Mounts**

   ```yaml
   volumes:
     - dynamodb-data:/home/dynamodblocal/data
     - minio-data:/data
   
   volumes:
     dynamodb-data:
     minio-data:
   ```

   **Pros**:
   - `docker compose down -v` will actually remove them
   - More portable across environments
   - Better permission handling
   
   **Cons**:
   - Harder to inspect data directly
   - Need `docker volume inspect` to find location

2. **Add Cleanup Script**

   Create `scripts/clean-local-db.sh`:
   ```bash
   #!/bin/bash
   docker compose down
   rm -rf dynamodb-data minio-data localstack-data
   echo "✓ All local data cleared"
   ```

3. **Add Data Validation on Startup**

   Check for corrupt data on API startup and log warnings.

4. **Add Health Check Endpoint**

   `GET /v1/admin/health/data` - Returns count of products without images/variants

---

## Testing

### Verification Steps

```bash
# 1. Check old product is gone
curl http://localhost:8080/v1/products/prod_1763239966434418382
# Expected: {"error":{"code":"NOT_FOUND"}}

# 2. Verify all products have images and variants
curl -s http://localhost:8080/v1/products | \
  python3 -c "import sys, json; data=json.load(sys.stdin); \
  [print(f'{p[\"title\"]}: {len(p[\"variants\"])} variants, {len(p.get(\"images\", []))} images') \
  for p in data['products']]"

# Expected:
# AI Generated Canvas Print: 3 variants, 1 images
# Custom AI Art Ceramic Mug: 2 variants, 1 images  
# AI Art Premium T-Shirt: 6 variants, 2 images
```

### Results ✅

- ✅ Old corrupt product removed
- ✅ All new products have variants
- ✅ All new products have images
- ✅ Validation prevents corrupt data

---

## Lessons Learned

1. **Bind mounts ≠ Named volumes**
   - `docker compose down -v` has different behavior
   - Always document data cleanup procedures

2. **Validate at write-time, not read-time**
   - Don't allow corrupt data to be saved
   - Add validation before database writes

3. **Make assumptions explicit**
   - Document storage mechanisms (bind vs volume)
   - Provide clear cleanup instructions

4. **Test data cleanup procedures**
   - Verify `-v` flag actually clears data
   - Include in development setup docs

---

## Action Items

### Completed ✅

- [x] Remove all corrupt data
- [x] Add product validation
- [x] Update .gitignore
- [x] Document root cause
- [x] Verify fix with tests

### Future Improvements

- [ ] Consider switching to named volumes
- [ ] Create `clean-local-db.sh` script
- [ ] Add data health check endpoint
- [ ] Add startup data validation
- [ ] Update README with cleanup instructions
- [ ] Add CI check for data directory presence

---

## Files Changed

1. `shop-api/internal/services/printful.go` - Added validateProductData()
2. `shop-api/.gitignore` - Added data directories
3. `shop-api/DATA_CORRUPTION_ROOT_CAUSE_ANALYSIS.md` - This document
4. `shop-api/DATABASE_VALIDATION_FIX.md` - Validation implementation details

---

## Summary

**Root Cause**: Docker Compose bind mounts to local directories were not removed by `docker compose down -v`

**Impact**: Corrupt data from November 15th persisted through database "clear" operation

**Resolution**: 
1. Manually removed bind-mounted directories
2. Added validation to prevent future corrupt data
3. Documented proper cleanup procedures
4. Updated .gitignore

**Status**: ✅ **RESOLVED** - All corrupt data removed, validation in place, procedures documented
