package testutil

import (
	"context"
	"os"

	"github.com/lemnispace/shop-api/internal/services"
	"github.com/lemnispace/shop-api/tests/mocks"
)

// S3ServiceProvider is an interface that can be implemented by both real and mock S3 services
type S3ServiceProvider interface {
	// Include all S3 service methods here
	services.S3Service
}

// SetupS3Environment configures environment variables for S3 tests
func SetupS3Environment() {
	// If environment variables aren't set, use default settings for local
	if os.Getenv("S3_ENDPOINT") == "" {
		// Default local development settings
		os.Setenv("S3_ENDPOINT", "http://localhost:9000")
		os.Setenv("S3_REGION", "us-east-1")
		os.Setenv("S3_ACCESS_KEY", "minioadmin")
		os.Setenv("S3_SECRET_KEY", "minioadmin")
	}
}

// ShouldUseRealS3 determines if tests should use real S3 or mocks
func ShouldUseRealS3() bool {
	// Always use mocks in CI to ensure tests pass reliably
	if os.Getenv("CI") == "true" {
		return false
	}
	
	// If explicitly told to use mocks, respect that
	if os.Getenv("USE_S3_MOCKS") == "true" {
		return false
	}

	// Try to create a real S3 service
	s3Service, err := services.NewS3Service()
	if err != nil {
		// Failed to create S3 service, use mocks
		return false
	}

	// Try a simple operation to verify S3 is accessible
	_, err = s3Service.BucketExists(context.Background(), "test-bucket")
	
	// If operation succeeds, we can use real S3
	return err == nil
}

// GetS3Service returns either a real S3 service or a mock based on environment
func GetS3Service() (S3ServiceProvider, bool) {
	// Check if we should use real S3
	useRealS3 := ShouldUseRealS3()
	
	if useRealS3 {
		// Set up the environment and create a real S3 service
		SetupS3Environment()
		s3Service, err := services.NewS3Service()
		
		// If there's an error, fall back to mock
		if err != nil {
			return mocks.NewMockS3Service(), false
		}
		
		return s3Service, true
	}
	
	// Return a pre-configured mock
	return mocks.NewMockS3Service(), false
} 