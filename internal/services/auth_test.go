package services

import (
	"context"
	"testing"
	"time"

	"github.com/lemnispace/shop-api/internal/models"
)

// mockCustomerService is a mock implementation of CustomerService for testing
type mockCustomerService struct {
	customers map[string]*models.Customer
}

func newMockCustomerService() *mockCustomerService {
	return &mockCustomerService{
		customers: make(map[string]*models.Customer),
	}
}

func (m *mockCustomerService) CreateCustomer(ctx context.Context, input *models.CustomerInput) (*models.Customer, error) {
	customer := &models.Customer{
		ID:               "cust_12345",
		Email:            input.Email,
		PasswordHash:     input.Password, // Mock - not actually hashed in test
		FirstName:        input.FirstName,
		LastName:         input.LastName,
		Phone:            input.Phone,
		AcceptsMarketing: input.AcceptsMarketing,
		DefaultAddress:   input.DefaultAddress,
		Addresses:        []models.Address{},
		Tags:             []string{},
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	m.customers[customer.Email] = customer
	return customer, nil
}

func (m *mockCustomerService) GetCustomer(ctx context.Context, customerID string) (*models.Customer, error) {
	for _, customer := range m.customers {
		if customer.ID == customerID {
			return customer, nil
		}
	}
	return nil, nil
}

func (m *mockCustomerService) GetCustomerByEmail(ctx context.Context, email string) (*models.Customer, error) {
	if customer, exists := m.customers[email]; exists {
		return customer, nil
	}
	return nil, nil
}

func (m *mockCustomerService) UpdateCustomer(ctx context.Context, customerID string, input *models.CustomerInput) error {
	return nil
}

func (m *mockCustomerService) DeleteCustomer(ctx context.Context, customerID string) error {
	return nil
}

func (m *mockCustomerService) ValidatePassword(ctx context.Context, email, password string) (*models.Customer, error) {
	customer, exists := m.customers[email]
	if !exists {
		return nil, nil
	}
	// Mock validation - just check if password matches the stored hash
	if customer.PasswordHash == password {
		return customer, nil
	}
	return nil, nil
}

func TestAuthService_Register(t *testing.T) {
	mockCustomer := newMockCustomerService()
	authService := NewAuthService(
		mockCustomer,
		"test-access-secret",
		"test-refresh-secret",
		15*time.Minute,
		7*24*time.Hour,
	)

	input := &models.CustomerInput{
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	response, err := authService.Register(context.Background(), input)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if response.AccessToken == "" {
		t.Error("Expected access token, got empty string")
	}

	if response.RefreshToken == "" {
		t.Error("Expected refresh token, got empty string")
	}

	if response.Customer == nil {
		t.Error("Expected customer in response")
	}

	if response.Customer.Email != input.Email {
		t.Errorf("Expected email %s, got %s", input.Email, response.Customer.Email)
	}
}

func TestAuthService_Login(t *testing.T) {
	mockCustomer := newMockCustomerService()
	authService := NewAuthService(
		mockCustomer,
		"test-access-secret",
		"test-refresh-secret",
		15*time.Minute,
		7*24*time.Hour,
	)

	// Register a customer first
	input := &models.CustomerInput{
		Email:    "test@example.com",
		Password: "password123",
	}
	mockCustomer.CreateCustomer(context.Background(), input)

	// Now try to login
	response, err := authService.Login(context.Background(), input.Email, input.Password)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if response.AccessToken == "" {
		t.Error("Expected access token, got empty string")
	}

	if response.RefreshToken == "" {
		t.Error("Expected refresh token, got empty string")
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	mockCustomer := newMockCustomerService()
	authService := NewAuthService(
		mockCustomer,
		"test-access-secret",
		"test-refresh-secret",
		15*time.Minute,
		7*24*time.Hour,
	).(*JWTAuthService)

	// Create a customer
	customer := &models.Customer{
		ID:    "cust_123",
		Email: "test@example.com",
	}

	// Generate tokens
	response, err := authService.generateTokenResponse(customer)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Validate the access token
	claims, err := authService.ValidateToken(response.AccessToken)
	if err != nil {
		t.Fatalf("Token validation failed: %v", err)
	}

	if claims.CustomerID != customer.ID {
		t.Errorf("Expected customer ID %s, got %s", customer.ID, claims.CustomerID)
	}

	if claims.Email != customer.Email {
		t.Errorf("Expected email %s, got %s", customer.Email, claims.Email)
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
	mockCustomer := newMockCustomerService()
	authService := NewAuthService(
		mockCustomer,
		"test-access-secret",
		"test-refresh-secret",
		15*time.Minute,
		7*24*time.Hour,
	).(*JWTAuthService)

	// Create a customer
	customer := &models.Customer{
		ID:    "cust_123",
		Email: "test@example.com",
	}
	mockCustomer.customers[customer.Email] = customer

	// Generate tokens
	response, err := authService.generateTokenResponse(customer)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Use refresh token to get new tokens
	newResponse, err := authService.RefreshToken(context.Background(), response.RefreshToken)
	if err != nil {
		t.Fatalf("Token refresh failed: %v", err)
	}

	if newResponse.AccessToken == "" {
		t.Error("Expected new access token, got empty string")
	}

	if newResponse.RefreshToken == "" {
		t.Error("Expected new refresh token, got empty string")
	}

	// Validate the new access token
	claims, err := authService.ValidateToken(newResponse.AccessToken)
	if err != nil {
		t.Fatalf("New token validation failed: %v", err)
	}

	if claims.CustomerID != customer.ID {
		t.Errorf("Expected customer ID %s, got %s", customer.ID, claims.CustomerID)
	}
}

func TestAuthService_InvalidToken(t *testing.T) {
	mockCustomer := newMockCustomerService()
	authService := NewAuthService(
		mockCustomer,
		"test-access-secret",
		"test-refresh-secret",
		15*time.Minute,
		7*24*time.Hour,
	)

	// Try to validate an invalid token
	_, err := authService.ValidateToken("invalid.token.string")
	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}
}
