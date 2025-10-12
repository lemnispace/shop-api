package utils

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// EncodeCursor encodes a DynamoDB LastEvaluatedKey into a string cursor
func EncodeCursor(lastEvaluatedKey map[string]types.AttributeValue) (string, error) {
	if len(lastEvaluatedKey) == 0 {
		return "", nil
	}

	// Convert DynamoDB AttributeValue map to a regular Go map
	var simpleMap map[string]interface{}
	err := attributevalue.UnmarshalMap(lastEvaluatedKey, &simpleMap)
	if err != nil {
		return "", err
	}

	// Marshal the simple map to JSON
	bytes, err := json.Marshal(simpleMap)
	if err != nil {
		return "", err
	}

	// Base64 encode the JSON
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// DecodeCursor decodes a string cursor into a DynamoDB key
func DecodeCursor(cursor string) (map[string]types.AttributeValue, error) {
	if cursor == "" {
		return nil, nil
	}

	// Base64 decode
	bytes, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON to simple map
	var simpleMap map[string]interface{}
	err = json.Unmarshal(bytes, &simpleMap)
	if err != nil {
		return nil, err
	}

	// Convert simple map back to DynamoDB AttributeValue map
	result, err := attributevalue.MarshalMap(simpleMap)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ExtractIDFromPK extracts an ID from a DynamoDB partition key
func ExtractIDFromPK(pk string) string {
	parts := strings.Split(pk, "#")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}
