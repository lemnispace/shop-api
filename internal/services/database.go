package services

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var DynamoDB *dynamodb.DynamoDB

func InitDynamoDB() error {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"), // Update region as needed
	})
	if err != nil {
		return err
	}
	DynamoDB = dynamodb.New(sess)
	return nil
}
