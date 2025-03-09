package utils

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// EncodeCursor encodes a DynamoDB LastEvaluatedKey into a string cursor
func EncodeCursor(lastEvaluatedKey map[string]types.AttributeValue) (string, error) {
	// For production, consider a more secure/efficient encoding method
	// This is a simplified example
	bytes, err := json.Marshal(lastEvaluatedKey)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// DecodeCursor decodes a string cursor into a DynamoDB key
func DecodeCursor(cursor string) (map[string]types.AttributeValue, error) {
	// For production, consider validation and error handling
	bytes, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, err
	}

	var result map[string]types.AttributeValue
	err = json.Unmarshal(bytes, &result)
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
