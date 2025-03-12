package utils

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// ConvertToDynamoDBAttributeValues converts a map of Go values to DynamoDB attribute values
func ConvertToDynamoDBAttributeValues(item map[string]interface{}) (map[string]types.AttributeValue, error) {
	result := make(map[string]types.AttributeValue)

	for key, value := range item {
		av, err := convertToAttributeValue(value)
		if err != nil {
			return nil, fmt.Errorf("failed to convert value for key %s: %w", key, err)
		}
		result[key] = av
	}

	return result, nil
}

// ConvertFromDynamoDBAttributeValues converts DynamoDB attribute values to a map of Go values
func ConvertFromDynamoDBAttributeValues(item map[string]types.AttributeValue) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	err := attributevalue.UnmarshalMap(item, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal DynamoDB item: %w", err)
	}

	return result, nil
}

// convertToAttributeValue converts a Go value to a DynamoDB attribute value
func convertToAttributeValue(value interface{}) (types.AttributeValue, error) {
	if value == nil {
		return &types.AttributeValueMemberNULL{Value: true}, nil
	}

	switch v := value.(type) {
	case string:
		return &types.AttributeValueMemberS{Value: v}, nil
	case int:
		return &types.AttributeValueMemberN{Value: strconv.Itoa(v)}, nil
	case int64:
		return &types.AttributeValueMemberN{Value: strconv.FormatInt(v, 10)}, nil
	case float64:
		return &types.AttributeValueMemberN{Value: strconv.FormatFloat(v, 'f', -1, 64)}, nil
	case bool:
		return &types.AttributeValueMemberBOOL{Value: v}, nil
	case []byte:
		return &types.AttributeValueMemberB{Value: v}, nil
	case time.Time:
		return &types.AttributeValueMemberS{Value: v.Format(time.RFC3339)}, nil
	case []string:
		ss := make([]string, len(v))
		for i, s := range v {
			ss[i] = s
		}
		return &types.AttributeValueMemberSS{Value: ss}, nil
	case []interface{}:
		list := make([]types.AttributeValue, len(v))
		for i, item := range v {
			av, err := convertToAttributeValue(item)
			if err != nil {
				return nil, err
			}
			list[i] = av
		}
		return &types.AttributeValueMemberL{Value: list}, nil
	case map[string]interface{}:
		m := make(map[string]types.AttributeValue)
		for k, val := range v {
			av, err := convertToAttributeValue(val)
			if err != nil {
				return nil, err
			}
			m[k] = av
		}
		return &types.AttributeValueMemberM{Value: m}, nil
	default:
		return nil, fmt.Errorf("unsupported type: %T", value)
	}
}

// QueryItems executes a query operation on a DynamoDB table and returns the results
func QueryItems(ctx context.Context, db *dynamodb.Client, tableName string, queryInput map[string]interface{}) ([]map[string]interface{}, error) {
	// Extract parameters from query input
	indexName, _ := queryInput["IndexName"].(string)
	keyConditionExpr, _ := queryInput["KeyConditionExpression"].(string)
	filterExpr, _ := queryInput["FilterExpression"].(string)

	// Extract expression attribute values
	exprAttributeValues := make(map[string]types.AttributeValue)
	if values, ok := queryInput["ExpressionAttributeValues"].(map[string]interface{}); ok {
		for k, v := range values {
			av, err := convertToAttributeValue(v)
			if err != nil {
				return nil, fmt.Errorf("failed to convert expression attribute value %s: %w", k, err)
			}
			exprAttributeValues[k] = av
		}
	}

	// Extract expression attribute names
	exprAttributeNames := make(map[string]string)
	if names, ok := queryInput["ExpressionAttributeNames"].(map[string]string); ok {
		exprAttributeNames = names
	}

	// Build the query input
	input := &dynamodb.QueryInput{
		TableName:                 aws.String(tableName),
		KeyConditionExpression:    aws.String(keyConditionExpr),
		ExpressionAttributeValues: exprAttributeValues,
	}

	// Add optional parameters
	if indexName != "" {
		input.IndexName = aws.String(indexName)
	}

	if filterExpr != "" {
		input.FilterExpression = aws.String(filterExpr)
	}

	if len(exprAttributeNames) > 0 {
		input.ExpressionAttributeNames = exprAttributeNames
	}

	// Execute the query
	result, err := db.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	// Convert the result to a slice of maps
	items := make([]map[string]interface{}, 0, len(result.Items))
	for _, item := range result.Items {
		m, err := ConvertFromDynamoDBAttributeValues(item)
		if err != nil {
			return nil, fmt.Errorf("failed to convert query result item: %w", err)
		}
		items = append(items, m)
	}

	return items, nil
}
