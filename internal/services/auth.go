package services

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lemnispace/shop-api/internal/models"
)

// AuthService defines the interface for authentication operations
type AuthService interface {
	Register(ctx context.Context, input *models.CustomerInput) (*models.CustomerLoginResponse, error)
	Login(ctx context.Context, email, password string) (*models.CustomerLoginResponse, error)
	ValidateToken(tokenString string) (*TokenClaims, error)
	RefreshToken(ctx context.Context, refreshTokenString string) (*models.CustomerLoginResponse, error)
}

// TokenClaims represents the JWT claims
type TokenClaims struct {
	CustomerID string `json:"customerId"`
	Email      string `json:"email"`
	jwt.RegisteredClaims
}

// JWTAuthService implements AuthService using JWT
type JWTAuthService struct {
	customerSvc       CustomerService
	accessTokenSecret  string
	refreshTokenSecret string
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
}

// NewAuthService creates a new authentication service
func NewAuthService(
	customerSvc CustomerService,
	accessTokenSecret string,
	refreshTokenSecret string,
	accessTokenExpiry time.Duration,
	refreshTokenExpiry time.Duration,
) AuthService {
	return &JWTAuthService{
		customerSvc:        customerSvc,
		accessTokenSecret:  accessTokenSecret,
		refreshTokenSecret: refreshTokenSecret,
		accessTokenExpiry:  accessTokenExpiry,
		refreshTokenExpiry: refreshTokenExpiry,
	}
}

// Register creates a new customer account
func (s *JWTAuthService) Register(ctx context.Context, input *models.CustomerInput) (*models.CustomerLoginResponse, error) {
	// Create customer
	customer, err := s.customerSvc.CreateCustomer(ctx, input)
	if err != nil {
		return nil, err
	}

	// Generate tokens
	return s.generateTokenResponse(customer)
}

// Login authenticates a customer
func (s *JWTAuthService) Login(ctx context.Context, email, password string) (*models.CustomerLoginResponse, error) {
	// Validate credentials
	customer, err := s.customerSvc.ValidatePassword(ctx, email, password)
	if err != nil {
		return nil, err
	}

	// Generate tokens
	return s.generateTokenResponse(customer)
}

// ValidateToken validates an access token
func (s *JWTAuthService) ValidateToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.accessTokenSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// RefreshToken generates a new access token from a refresh token
func (s *JWTAuthService) RefreshToken(ctx context.Context, refreshTokenString string) (*models.CustomerLoginResponse, error) {
	// Parse and validate refresh token
	token, err := jwt.ParseWithClaims(refreshTokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.refreshTokenSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid refresh token claims")
	}

	// Get customer
	customer, err := s.customerSvc.GetCustomer(ctx, claims.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("customer not found")
	}

	// Generate new tokens
	return s.generateTokenResponse(customer)
}

// generateTokenResponse creates access and refresh tokens
func (s *JWTAuthService) generateTokenResponse(customer *models.Customer) (*models.CustomerLoginResponse, error) {
	now := time.Now()
	accessExpiry := now.Add(s.accessTokenExpiry)
	refreshExpiry := now.Add(s.refreshTokenExpiry)

	// Create access token
	accessClaims := TokenClaims{
		CustomerID: customer.ID,
		Email:      customer.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(s.accessTokenSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Create refresh token
	refreshClaims := TokenClaims{
		CustomerID: customer.ID,
		Email:      customer.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(s.refreshTokenSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &models.CustomerLoginResponse{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    accessExpiry,
		Customer:     customer,
	}, nil
}
