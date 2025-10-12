#!/bin/bash
# Create DynamoDB table using curl and the AWS DynamoDB API

ENDPOINT="http://localhost:8000"

# Create table using AWS Signature Version 4
curl -X POST "$ENDPOINT/" \
  -H "Content-Type: application/x-amz-json-1.0" \
  -H "X-Amz-Target: DynamoDB_20120810.CreateTable" \
  -d '{
    "TableName": "ShopAPI",
    "KeySchema": [
      {"AttributeName": "PK", "KeyType": "HASH"},
      {"AttributeName": "SK", "KeyType": "RANGE"}
    ],
    "AttributeDefinitions": [
      {"AttributeName": "PK", "AttributeType": "S"},
      {"AttributeName": "SK", "AttributeType": "S"},
      {"AttributeName": "GSI1PK", "AttributeType": "S"},
      {"AttributeName": "GSI1SK", "AttributeType": "S"},
      {"AttributeName": "GSI2PK", "AttributeType": "S"},
      {"AttributeName": "GSI2SK", "AttributeType": "S"}
    ],
    "GlobalSecondaryIndexes": [
      {
        "IndexName": "GSI1",
        "KeySchema": [
          {"AttributeName": "GSI1PK", "KeyType": "HASH"},
          {"AttributeName": "GSI1SK", "KeyType": "RANGE"}
        ],
        "Projection": {"ProjectionType": "ALL"},
        "ProvisionedThroughput": {"ReadCapacityUnits": 5, "WriteCapacityUnits": 5}
      },
      {
        "IndexName": "GSI2",
        "KeySchema": [
          {"AttributeName": "GSI2PK", "KeyType": "HASH"},
          {"AttributeName": "GSI2SK", "KeyType": "RANGE"}
        ],
        "Projection": {"ProjectionType": "ALL"},
        "ProvisionedThroughput": {"ReadCapacityUnits": 5, "WriteCapacityUnits": 5}
      }
    ],
    "ProvisionedThroughput": {
      "ReadCapacityUnits": 5,
      "WriteCapacityUnits": 5
    }
  }'

echo -e "\n\nTable created!"
