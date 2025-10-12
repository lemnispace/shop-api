#!/bin/bash
# Initialize local DynamoDB with the ShopAPI table

set -e

ENDPOINT="http://localhost:8000"
TABLE_NAME="ShopAPI"

echo "Creating DynamoDB table: $TABLE_NAME"

aws dynamodb create-table \
    --endpoint-url $ENDPOINT \
    --table-name $TABLE_NAME \
    --attribute-definitions \
        AttributeName=PK,AttributeType=S \
        AttributeName=SK,AttributeType=S \
        AttributeName=GSI1PK,AttributeType=S \
        AttributeName=GSI1SK,AttributeType=S \
        AttributeName=GSI2PK,AttributeType=S \
        AttributeName=GSI2SK,AttributeType=S \
    --key-schema \
        AttributeName=PK,KeyType=HASH \
        AttributeName=SK,KeyType=RANGE \
    --global-secondary-indexes \
        "[
            {
                \"IndexName\": \"GSI1\",
                \"KeySchema\": [
                    {\"AttributeName\":\"GSI1PK\",\"KeyType\":\"HASH\"},
                    {\"AttributeName\":\"GSI1SK\",\"KeyType\":\"RANGE\"}
                ],
                \"Projection\": {\"ProjectionType\":\"ALL\"},
                \"ProvisionedThroughput\": {\"ReadCapacityUnits\":5,\"WriteCapacityUnits\":5}
            },
            {
                \"IndexName\": \"GSI2\",
                \"KeySchema\": [
                    {\"AttributeName\":\"GSI2PK\",\"KeyType\":\"HASH\"},
                    {\"AttributeName\":\"GSI2SK\",\"KeyType\":\"RANGE\"}
                ],
                \"Projection\": {\"ProjectionType\":\"ALL\"},
                \"ProvisionedThroughput\": {\"ReadCapacityUnits\":5,\"WriteCapacityUnits\":5}
            }
        ]" \
    --provisioned-throughput \
        ReadCapacityUnits=5,WriteCapacityUnits=5 \
    --region us-east-1 \
    2>/dev/null || echo "Table $TABLE_NAME already exists"

echo "DynamoDB table initialized successfully!"
echo ""
echo "You can verify with:"
echo "  aws dynamodb list-tables --endpoint-url $ENDPOINT --region us-east-1"
