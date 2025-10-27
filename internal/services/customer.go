package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/lemnispace/shop-api/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// CustomerService defines the interface for customer operations
type CustomerService interface {
	CreateCustomer(ctx context.Context, input *models.CustomerInput) (*models.Customer, error)
	GetCustomer(ctx context.Context, customerID string) (*models.Customer, error)
	GetCustomerByEmail(ctx context.Context, email string) (*models.Customer, error)
	UpdateCustomer(ctx context.Context, customerID string, input *models.CustomerInput) error
	DeleteCustomer(ctx context.Context, customerID string) error
	ValidatePassword(ctx context.Context, email, password string) (*models.Customer, error)
}

// DynamoDBCustomerService implements CustomerService using DynamoDB
type DynamoDBCustomerService struct {
	client    *dynamodb.Client
	tableName string
}

// NewCustomerService creates a new customer service
func NewCustomerService(client *dynamodb.Client, tableName string) CustomerService {
	log.Printf("Initializing DynamoDB Customer Service with table: %s", tableName)
	return &DynamoDBCustomerService{
		client:    client,
		tableName: tableName,
	}
}

// CreateCustomer creates a new customer with hashed password
func (s *DynamoDBCustomerService) CreateCustomer(ctx context.Context, input *models.CustomerInput) (*models.Customer, error) {
	// Check if email already exists
	existing, err := s.GetCustomerByEmail(ctx, input.Email)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("customer with email %s already exists", input.Email)
	}

	// Generate customer ID
	customerID := fmt.Sprintf("cust_%d", time.Now().UnixNano())

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	customer := &models.Customer{
		ID:               customerID,
		Email:            input.Email,
		PasswordHash:     string(hashedPassword),
		FirstName:        input.FirstName,
		LastName:         input.LastName,
		Phone:            input.Phone,
		AcceptsMarketing: input.AcceptsMarketing,
		DefaultAddress:   input.DefaultAddress,
		Tags:             []string{},
		Addresses:        []models.Address{},
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Add default address to addresses if provided
	if input.DefaultAddress.Address1 != "" {
		customer.Addresses = []models.Address{input.DefaultAddress}
	}

	// Store customer metadata
	av, err := attributevalue.MarshalMap(customer)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal customer: %w", err)
	}

	av["PK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("CUSTOMER#%s", customerID)}
	av["SK"] = &types.AttributeValueMemberS{Value: "METADATA"}
	av["GSI1PK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("EMAIL#%s", input.Email)}
	av["GSI1SK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("CUSTOMER#%s", customerID)}

	// TODO(concurrency): Replace the read-then-write uniqueness check with a conditional PutItem
	// (attribute_not_exists(PK)) or a transactional write to avoid duplicate emails under load.
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      av,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	log.Printf("Created customer: %s with email: %s", customerID, input.Email)
	return customer, nil
}

// GetCustomer retrieves a customer by ID
func (s *DynamoDBCustomerService) GetCustomer(ctx context.Context, customerID string) (*models.Customer, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("CUSTOMER#%s", customerID)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("customer not found")
	}

	var customer models.Customer
	if err := attributevalue.UnmarshalMap(result.Item, &customer); err != nil {
		return nil, fmt.Errorf("failed to unmarshal customer: %w", err)
	}

	return &customer, nil
}

// GetCustomerByEmail retrieves a customer by email using GSI1
func (s *DynamoDBCustomerService) GetCustomerByEmail(ctx context.Context, email string) (*models.Customer, error) {
	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :email"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":email": &types.AttributeValueMemberS{Value: fmt.Sprintf("EMAIL#%s", email)},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query customer by email: %w", err)
	}

	if len(result.Items) == 0 {
		return nil, fmt.Errorf("customer not found")
	}

	var customer models.Customer
	if err := attributevalue.UnmarshalMap(result.Items[0], &customer); err != nil {
		return nil, fmt.Errorf("failed to unmarshal customer: %w", err)
	}

	return &customer, nil
}

// UpdateCustomer updates an existing customer
func (s *DynamoDBCustomerService) UpdateCustomer(ctx context.Context, customerID string, input *models.CustomerInput) error {
	// Get existing customer
	existing, err := s.GetCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	// Update fields
	existing.FirstName = input.FirstName
	existing.LastName = input.LastName
	existing.Phone = input.Phone
	existing.AcceptsMarketing = input.AcceptsMarketing
	existing.DefaultAddress = input.DefaultAddress
	existing.UpdatedAt = time.Now()

	// Hash new password if provided
	if input.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		existing.PasswordHash = string(hashedPassword)
	}

	// Update addresses
	if input.DefaultAddress.Address1 != "" {
		found := false
		for i, addr := range existing.Addresses {
			if addr.Address1 == input.DefaultAddress.Address1 {
				existing.Addresses[i] = input.DefaultAddress
				found = true
				break
			}
		}
		if !found {
			existing.Addresses = append(existing.Addresses, input.DefaultAddress)
		}
	}

	// Marshal and update
	av, err := attributevalue.MarshalMap(existing)
	if err != nil {
		return fmt.Errorf("failed to marshal customer: %w", err)
	}

	av["PK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("CUSTOMER#%s", customerID)}
	av["SK"] = &types.AttributeValueMemberS{Value: "METADATA"}
	av["GSI1PK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("EMAIL#%s", existing.Email)}
	av["GSI1SK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("CUSTOMER#%s", customerID)}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to update customer: %w", err)
	}

	log.Printf("Updated customer: %s", customerID)
	return nil
}

// DeleteCustomer deletes a customer
func (s *DynamoDBCustomerService) DeleteCustomer(ctx context.Context, customerID string) error {
	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("CUSTOMER#%s", customerID)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete customer: %w", err)
	}

	log.Printf("Deleted customer: %s", customerID)
	return nil
}

// ValidatePassword validates a customer's password
func (s *DynamoDBCustomerService) ValidatePassword(ctx context.Context, email, password string) (*models.Customer, error) {
	customer, err := s.GetCustomerByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(customer.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return customer, nil
}
