#!/bin/bash
# Clean Local Database Script
# 
# This script properly clears all local development data including:
# - DynamoDB tables
# - MinIO/S3 buckets
# - LocalStack data
#
# Run this when you need to start with a completely fresh database.

set -e

echo "=== Clean Local Database ==="
echo ""
echo "⚠️  WARNING: This will delete ALL local development data!"
echo "   - DynamoDB tables"
echo "   - MinIO/S3 buckets"  
echo "   - LocalStack resources"
echo ""

# Ask for confirmation unless --force flag is provided
if [ "$1" != "--force" ]; then
    read -p "Are you sure you want to continue? (yes/no): " confirm
    if [ "$confirm" != "yes" ]; then
        echo "Aborted."
        exit 0
    fi
fi

echo ""
echo "Step 1: Stopping Docker containers..."
docker compose down

echo ""
echo "Step 2: Removing data directories..."
rm -rf dynamodb-data minio-data localstack-data
echo "  ✓ dynamodb-data/"
echo "  ✓ minio-data/"
echo "  ✓ localstack-data/"

echo ""
echo "Step 3: Removing Docker volumes (if any)..."
docker volume rm shop-api_dynamodb-data shop-api_minio-data 2>/dev/null || echo "  (no named volumes to remove)"

echo ""
echo "✓ All local data cleared successfully!"
echo ""
echo "Next steps:"
echo "  1. Start services:    docker compose up -d"
echo "  2. Populate data:     python3 scripts/populate-mock-data.py"
echo "  3. Or import Printful: bash scripts/populate-local-db.sh"
echo ""
