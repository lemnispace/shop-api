# Testing Architecture for Shop API

This directory contains tests for the Shop API. The testing architecture is designed to be robust and work reliably in both local development environments and CI pipelines.

## Directory Structure

- `api/` - API integration tests
- `unit/` - Unit tests for individual components
- `mocks/` - Mock implementations for external dependencies
- `testutil/` - Shared test utilities and helpers

## S3 Testing Strategy

The testing framework supports both real and mock S3 interactions:

1. **Local Development**: Tests use a real S3 service (typically MinIO) when available. The MinIO server should be running locally at `http://localhost:9000` with the default credentials (minioadmin/minioadmin).

2. **CI Environment**: Tests automatically use mocks in CI environments to ensure tests pass reliably without requiring external services.

3. **Fallback to Mocks**: When real S3 is not available even in local development, tests automatically fall back to using mocks.

## Using the Test Utilities

### Environment Setup

The `testutil` package provides functions to correctly set up the test environment:

```go
import "github.com/lemnispace/shop-api/tests/testutil"

func TestYourFeature(t *testing.T) {
    // Set up environment variables for S3
    testutil.SetupS3Environment()
    
    // Rest of your test...
}
```

### Getting an S3 Service

To get the appropriate S3 service (real or mock) based on the environment:

```go
import "github.com/lemnispace/shop-api/tests/testutil"

func TestWithS3(t *testing.T) {
    // Get an S3 service - will be real if possible, mock otherwise
    s3Service, isReal := testutil.GetS3Service()
    
    // You can check if it's using a real implementation
    if isReal {
        // Do something specific for real S3
    } else {
        // Set up mock expectations for the mock S3
        mockS3 := s3Service.(*mocks.MockS3Service)
        // Configure mock...
    }
    
    // Use s3Service for your tests...
}
```

## Using Mock Implementations

### Basic Mock with Default Settings

```go
import "github.com/lemnispace/shop-api/tests/mocks"

// Get a mock with all methods returning success
mockS3 := mocks.NewMockS3Service()
```

### Specific Mock for Targeted Tests

```go
import "github.com/lemnispace/shop-api/tests/mocks"

// Create a mock configured for a specific bucket and key
bucketName := "test-bucket"
objectKey := "path/to/image.jpg"
fileData := []byte("test file content") 
contentType := "image/jpeg"

mockS3 := mocks.NewSpecificMockS3Service(bucketName, objectKey, fileData, contentType)
```

### Custom Mock Expectations

For more complex test scenarios, you can create a mock and configure it with your specific expectations:

```go
import (
    "github.com/lemnispace/shop-api/tests/mocks"
    "github.com/stretchr/testify/mock"
)

mockS3 := new(mocks.MockS3Service)
mockS3.On("BucketExists", mock.Anything, "missing-bucket").Return(false, nil)
mockS3.On("UploadFile", mock.Anything, "test-bucket", "error.jpg", mock.Anything, mock.Anything).Return(errors.New("upload failed"))
```

## Best Practices

1. **No test skipping**: Tests should always run, in both local and CI environments.

2. **Deterministic tests**: Tests should be deterministic and not depend on external state.

3. **Avoid hard-coded credentials**: Use the provided test utilities to set up environment variables.

4. **Clean up test resources**: Ensure any created test resources are cleaned up to avoid conflicts between tests.

5. **Use descriptive test names**: Test names should clearly describe what is being tested.

6. **Meaningful assertions**: Ensure your test assertions provide meaningful messages when they fail.

## Test Structure

- `api/` - API integration tests for testing the full HTTP endpoints
- `unit/` - Unit tests for testing individual components in isolation
- `integration/` - Integration tests for testing interactions between components

## Running Tests

You can run the tests using the following Make commands:

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run API tests only
make test-api

# Run a specific test by name pattern
make test-pattern PATTERN=TestCollectionProducts

# Run tests with race detection
make test-race

# Run tests and generate coverage report
make test-coverage
```

Or you can use the provided script directly:

```bash
./scripts/run-tests.sh ./tests/...
```

## Test Implementation Strategy

1. **API Tests**: These test the full HTTP request/response cycle by creating a test server with the actual handlers and routes. They verify that the API endpoints work as expected.

2. **Unit Tests**: These test individual components (like services) in isolation. They verify the business logic works correctly.

3. **Integration Tests**: These test the interactions between components, such as how the handlers use the services.

## Troubleshooting Tests

If tests fail, you can:

1. Check the logs for detailed error messages
2. Try running a specific test to isolate the issue: `make test-pattern PATTERN=TestCollectionProducts`
3. Verify that DynamoDB is running: `docker ps | grep dynamodb-local`
4. Reset the test environment: `make dynamo-stop && make dynamo-local && make dynamo-init`

## GitHub Actions Integration

The tests are automatically run in GitHub Actions CI workflow whenever changes are pushed to the main branches. The workflow is defined in `.github/workflows/ci.yml`.
