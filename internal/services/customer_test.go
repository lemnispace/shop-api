package services

import (
	"context"
	"testing"
	"time"

	"github.com/lemnispace/shop-api/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestCustomerService_CreateCustomer(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	customerService := NewCustomerService(client, tableName)

	input := &models.CustomerInput{
		Email:            "test@example.com",
		Password:         "SecurePassword123",
		FirstName:        "John",
		LastName:         "Doe",
		Phone:            "+1234567890",
		AcceptsMarketing: true,
	}

	customer, err := customerService.CreateCustomer(context.Background(), input)
	require.NoError(t, err)
	assert.NotEmpty(t, customer.ID)
	assert.Equal(t, input.Email, customer.Email)
	assert.Equal(t, input.FirstName, customer.FirstName)
	assert.Equal(t, input.LastName, customer.LastName)
	assert.Equal(t, input.Phone, customer.Phone)
	assert.Equal(t, input.AcceptsMarketing, customer.AcceptsMarketing)
	assert.NotEmpty(t, customer.PasswordHash)
	assert.NotEqual(t, input.Password, customer.PasswordHash) // Password should be hashed
	assert.NotZero(t, customer.CreatedAt)
	assert.NotZero(t, customer.UpdatedAt)

	// Verify password is properly hashed
	err = bcrypt.CompareHashAndPassword([]byte(customer.PasswordHash), []byte(input.Password))
	assert.NoError(t, err, "Password hash should match original password")
}

func TestCustomerService_CreateCustomer_WithAddress(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	customerService := NewCustomerService(client, tableName)

	input := &models.CustomerInput{
		Email:     "test@example.com",
		Password:  "SecurePassword123",
		FirstName: "John",
		LastName:  "Doe",
		DefaultAddress: models.Address{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "123 Main St",
			Address2:  "Apt 4B",
			City:      "New York",
			Province:  "NY",
			Country:   "US",
			Zip:       "10001",
			Phone:     "+1234567890",
		},
	}

	customer, err := customerService.CreateCustomer(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, input.DefaultAddress.Address1, customer.DefaultAddress.Address1)
	assert.Equal(t, input.DefaultAddress.City, customer.DefaultAddress.City)
	assert.Len(t, customer.Addresses, 1)
	assert.Equal(t, input.DefaultAddress.Address1, customer.Addresses[0].Address1)
}

func TestCustomerService_CreateCustomer_DuplicateEmail(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	customerService := NewCustomerService(client, tableName)

	input := &models.CustomerInput{
		Email:     "test@example.com",
		Password:  "SecurePassword123",
		FirstName: "John",
		LastName:  "Doe",
	}

	// Create first customer
	_, err := customerService.CreateCustomer(context.Background(), input)
	require.NoError(t, err)

	// Try to create second customer with same email
	_, err = customerService.CreateCustomer(context.Background(), input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestCustomerService_GetCustomer(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	customerService := NewCustomerService(client, tableName)

	// Create a customer
	input := &models.CustomerInput{
		Email:     "test@example.com",
		Password:  "SecurePassword123",
		FirstName: "John",
		LastName:  "Doe",
	}

	created, err := customerService.CreateCustomer(context.Background(), input)
	require.NoError(t, err)

	// Get the customer
	retrieved, err := customerService.GetCustomer(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, created.Email, retrieved.Email)
	assert.Equal(t, created.FirstName, retrieved.FirstName)
	assert.Equal(t, created.LastName, retrieved.LastName)
}

func TestCustomerService_GetCustomer_NotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	customerService := NewCustomerService(client, tableName)

	_, err := customerService.GetCustomer(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCustomerService_GetCustomerByEmail(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	customerService := NewCustomerService(client, tableName)

	// Create a customer
	input := &models.CustomerInput{
		Email:     "test@example.com",
		Password:  "SecurePassword123",
		FirstName: "John",
		LastName:  "Doe",
	}

	created, err := customerService.CreateCustomer(context.Background(), input)
	require.NoError(t, err)

	// Get customer by email
	retrieved, err := customerService.GetCustomerByEmail(context.Background(), input.Email)
	require.NoError(t, err)
	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, created.Email, retrieved.Email)
}

func TestCustomerService_GetCustomerByEmail_NotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	customerService := NewCustomerService(client, tableName)

	_, err := customerService.GetCustomerByEmail(context.Background(), "nonexistent@example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCustomerService_UpdateCustomer(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	customerService := NewCustomerService(client, tableName)

	// Create a customer
	input := &models.CustomerInput{
		Email:     "test@example.com",
		Password:  "SecurePassword123",
		FirstName: "John",
		LastName:  "Doe",
		Phone:     "+1234567890",
	}

	created, err := customerService.CreateCustomer(context.Background(), input)
	require.NoError(t, err)

	// Wait to ensure different timestamp
	time.Sleep(10 * time.Millisecond)

	// Update the customer
	updateInput := &models.CustomerInput{
		Email:            created.Email, // Email cannot be changed
		FirstName:        "Jane",
		LastName:         "Smith",
		Phone:            "+9876543210",
		AcceptsMarketing: true,
	}

	err = customerService.UpdateCustomer(context.Background(), created.ID, updateInput)
	require.NoError(t, err)

	// Verify update
	updated, err := customerService.GetCustomer(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Jane", updated.FirstName)
	assert.Equal(t, "Smith", updated.LastName)
	assert.Equal(t, "+9876543210", updated.Phone)
	assert.True(t, updated.AcceptsMarketing)
	assert.True(t, updated.UpdatedAt.After(created.UpdatedAt))
}

func TestCustomerService_UpdateCustomer_Password(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	customerService := NewCustomerService(client, tableName)

	// Create a customer
	input := &models.CustomerInput{
		Email:     "test@example.com",
		Password:  "OldPassword123",
		FirstName: "John",
		LastName:  "Doe",
	}

	created, err := customerService.CreateCustomer(context.Background(), input)
	require.NoError(t, err)
	oldPasswordHash := created.PasswordHash

	// Update password
	updateInput := &models.CustomerInput{
		Email:     created.Email,
		Password:  "NewPassword456",
		FirstName: created.FirstName,
		LastName:  created.LastName,
	}

	err = customerService.UpdateCustomer(context.Background(), created.ID, updateInput)
	require.NoError(t, err)

	// Verify new password
	updated, err := customerService.GetCustomer(context.Background(), created.ID)
	require.NoError(t, err)
	assert.NotEqual(t, oldPasswordHash, updated.PasswordHash)

	// Verify old password no longer works
	err = bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte("OldPassword123"))
	assert.Error(t, err)

	// Verify new password works
	err = bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte("NewPassword456"))
	assert.NoError(t, err)
}

func TestCustomerService_UpdateCustomer_AddAddress(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	customerService := NewCustomerService(client, tableName)

	// Create a customer without address
	input := &models.CustomerInput{
		Email:     "test@example.com",
		Password:  "SecurePassword123",
		FirstName: "John",
		LastName:  "Doe",
	}

	created, err := customerService.CreateCustomer(context.Background(), input)
	require.NoError(t, err)

	// Add address via update
	updateInput := &models.CustomerInput{
		Email:     created.Email,
		FirstName: created.FirstName,
		LastName:  created.LastName,
		DefaultAddress: models.Address{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "456 Oak Ave",
			City:      "Boston",
			Province:  "MA",
			Country:   "US",
			Zip:       "02101",
		},
	}

	err = customerService.UpdateCustomer(context.Background(), created.ID, updateInput)
	require.NoError(t, err)

	// Verify address was added
	updated, err := customerService.GetCustomer(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, "456 Oak Ave", updated.DefaultAddress.Address1)
	assert.Len(t, updated.Addresses, 1)
}

func TestCustomerService_UpdateCustomer_UpdateExistingAddress(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	customerService := NewCustomerService(client, tableName)

	// Create a customer with address
	input := &models.CustomerInput{
		Email:     "test@example.com",
		Password:  "SecurePassword123",
		FirstName: "John",
		LastName:  "Doe",
		DefaultAddress: models.Address{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "123 Main St",
			City:      "New York",
			Province:  "NY",
			Country:   "US",
			Zip:       "10001",
		},
	}

	created, err := customerService.CreateCustomer(context.Background(), input)
	require.NoError(t, err)

	// Update the same address (matching Address1)
	updateInput := &models.CustomerInput{
		Email:     created.Email,
		FirstName: created.FirstName,
		LastName:  created.LastName,
		DefaultAddress: models.Address{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "123 Main St", // Same address
			City:      "New York",
			Province:  "NY",
			Country:   "US",
			Zip:       "10002", // Different zip
		},
	}

	err = customerService.UpdateCustomer(context.Background(), created.ID, updateInput)
	require.NoError(t, err)

	// Verify address was updated, not added
	updated, err := customerService.GetCustomer(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Len(t, updated.Addresses, 1)
	assert.Equal(t, "10002", updated.Addresses[0].Zip)
}

func TestCustomerService_UpdateCustomer_NotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	customerService := NewCustomerService(client, tableName)

	updateInput := &models.CustomerInput{
		Email:     "test@example.com",
		FirstName: "John",
		LastName:  "Doe",
	}

	err := customerService.UpdateCustomer(context.Background(), "nonexistent", updateInput)
	assert.Error(t, err)
}

func TestCustomerService_DeleteCustomer(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	customerService := NewCustomerService(client, tableName)

	// Create a customer
	input := &models.CustomerInput{
		Email:     "test@example.com",
		Password:  "SecurePassword123",
		FirstName: "John",
		LastName:  "Doe",
	}

	created, err := customerService.CreateCustomer(context.Background(), input)
	require.NoError(t, err)

	// Delete the customer
	err = customerService.DeleteCustomer(context.Background(), created.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = customerService.GetCustomer(context.Background(), created.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCustomerService_ValidatePassword_Success(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	customerService := NewCustomerService(client, tableName)

	// Create a customer
	input := &models.CustomerInput{
		Email:     "test@example.com",
		Password:  "SecurePassword123",
		FirstName: "John",
		LastName:  "Doe",
	}

	created, err := customerService.CreateCustomer(context.Background(), input)
	require.NoError(t, err)

	// Validate correct password
	customer, err := customerService.ValidatePassword(context.Background(), input.Email, input.Password)
	require.NoError(t, err)
	assert.Equal(t, created.ID, customer.ID)
	assert.Equal(t, created.Email, customer.Email)
}

func TestCustomerService_ValidatePassword_WrongPassword(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	customerService := NewCustomerService(client, tableName)

	// Create a customer
	input := &models.CustomerInput{
		Email:     "test@example.com",
		Password:  "SecurePassword123",
		FirstName: "John",
		LastName:  "Doe",
	}

	_, err := customerService.CreateCustomer(context.Background(), input)
	require.NoError(t, err)

	// Try to validate with wrong password
	_, err = customerService.ValidatePassword(context.Background(), input.Email, "WrongPassword")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestCustomerService_ValidatePassword_EmailNotFound(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	customerService := NewCustomerService(client, tableName)

	// Try to validate with non-existent email
	_, err := customerService.ValidatePassword(context.Background(), "nonexistent@example.com", "SomePassword")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestCustomerService_PasswordHashing(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	customerService := NewCustomerService(client, tableName)

	// Create two customers with the same password
	input1 := &models.CustomerInput{
		Email:     "user1@example.com",
		Password:  "SamePassword123",
		FirstName: "User",
		LastName:  "One",
	}

	input2 := &models.CustomerInput{
		Email:     "user2@example.com",
		Password:  "SamePassword123",
		FirstName: "User",
		LastName:  "Two",
	}

	customer1, err := customerService.CreateCustomer(context.Background(), input1)
	require.NoError(t, err)

	customer2, err := customerService.CreateCustomer(context.Background(), input2)
	require.NoError(t, err)

	// Verify hashes are different (bcrypt uses salt)
	assert.NotEqual(t, customer1.PasswordHash, customer2.PasswordHash)

	// Verify both hashes validate correctly
	err = bcrypt.CompareHashAndPassword([]byte(customer1.PasswordHash), []byte("SamePassword123"))
	assert.NoError(t, err)

	err = bcrypt.CompareHashAndPassword([]byte(customer2.PasswordHash), []byte("SamePassword123"))
	assert.NoError(t, err)
}

func TestCustomerService_MultipleAddresses(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	customerService := NewCustomerService(client, tableName)

	// Create a customer with initial address
	input := &models.CustomerInput{
		Email:     "test@example.com",
		Password:  "SecurePassword123",
		FirstName: "John",
		LastName:  "Doe",
		DefaultAddress: models.Address{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "123 Main St",
			City:      "New York",
			Country:   "US",
			Zip:       "10001",
		},
	}

	created, err := customerService.CreateCustomer(context.Background(), input)
	require.NoError(t, err)
	assert.Len(t, created.Addresses, 1)

	// Add a second address
	updateInput := &models.CustomerInput{
		Email:     created.Email,
		FirstName: created.FirstName,
		LastName:  created.LastName,
		DefaultAddress: models.Address{
			FirstName: "John",
			LastName:  "Doe",
			Address1:  "456 Oak Ave",
			City:      "Boston",
			Country:   "US",
			Zip:       "02101",
		},
	}

	err = customerService.UpdateCustomer(context.Background(), created.ID, updateInput)
	require.NoError(t, err)

	// Verify both addresses exist
	updated, err := customerService.GetCustomer(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Len(t, updated.Addresses, 2)
}

func TestCustomerService_EmptyOptionalFields(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	customerService := NewCustomerService(client, tableName)

	// Create customer with only required fields
	input := &models.CustomerInput{
		Email:    "minimal@example.com",
		Password: "SecurePassword123",
	}

	customer, err := customerService.CreateCustomer(context.Background(), input)
	require.NoError(t, err)
	assert.NotEmpty(t, customer.ID)
	assert.Equal(t, input.Email, customer.Email)
	assert.Empty(t, customer.FirstName)
	assert.Empty(t, customer.LastName)
	assert.Empty(t, customer.Phone)
	assert.False(t, customer.AcceptsMarketing)
	assert.Empty(t, customer.DefaultAddress.Address1)
	assert.Empty(t, customer.Addresses)
}

func TestCustomerService_UpdateCustomer_NoPasswordChange(t *testing.T) {
	client, tableName := setupTestDynamoDB(t)
	defer cleanupTestTable(t, client, tableName)

	customerService := NewCustomerService(client, tableName)

	// Create a customer
	input := &models.CustomerInput{
		Email:     "test@example.com",
		Password:  "OriginalPassword123",
		FirstName: "John",
		LastName:  "Doe",
	}

	created, err := customerService.CreateCustomer(context.Background(), input)
	require.NoError(t, err)
	originalPasswordHash := created.PasswordHash

	// Update without password
	updateInput := &models.CustomerInput{
		Email:     created.Email,
		FirstName: "Jane",
		LastName:  "Smith",
		// Password is empty, should not change
	}

	err = customerService.UpdateCustomer(context.Background(), created.ID, updateInput)
	require.NoError(t, err)

	// Verify password hash unchanged
	updated, err := customerService.GetCustomer(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, originalPasswordHash, updated.PasswordHash)

	// Verify original password still works
	err = bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte("OriginalPassword123"))
	assert.NoError(t, err)
}
