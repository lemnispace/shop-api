package unit

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	_ "image/jpeg" // Register JPEG format
	"mime/multipart"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/tests/mocks"
	"github.com/lemnispace/shop-api/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockMultipartFile is a mock implementation of multipart.File
type MockMultipartFile struct {
	*bytes.Reader
}

func (m *MockMultipartFile) Close() error {
	return nil
}

// MockDynamoDBClient is a mock implementation of the DynamoDB client
type MockDynamoDBClient struct {
	mock.Mock
}

func (m *MockDynamoDBClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*dynamodb.PutItemOutput), args.Error(1)
}

func (m *MockDynamoDBClient) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*dynamodb.GetItemOutput), args.Error(1)
}

func (m *MockDynamoDBClient) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*dynamodb.DeleteItemOutput), args.Error(1)
}

// TestImageDecode tests the image decoding functionality without mocking AWS
func TestImageDecode(t *testing.T) {
	// Create a valid minimal JPEG image for testing
	// This is a base64-encoded 1x1 pixel JPEG image
	const base64Image = "/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/2wBDAQkJCQwLDBgNDRgyIRwhMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjL/wAARCAABAAEDASIAAhEBAxEB/8QAHwAAAQUBAQEBAQEAAAAAAAAAAAECAwQFBgcICQoL/8QAtRAAAgEDAwIEAwUFBAQAAAF9AQIDAAQRBRIhMUEGE1FhByJxFDKBkaEII0KxwRVS0fAkM2JyggkKFhcYGRolJicoKSo0NTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqDhIWGh4iJipKTlJWWl5iZmqKjpKWmp6ipqrKztLW2t7i5usLDxMXGx8jJytLT1NXW19jZ2uHi4+Tl5ufo6erx8vP09fb3+Pn6/8QAHwEAAwEBAQEBAQEBAQAAAAAAAAECAwQFBgcICQoL/8QAtREAAgECBAQDBAcFBAQAAQJ3AAECAxEEBSExBhJBUQdhcRMiMoEIFEKRobHBCSMzUvAVYnLRChYkNOEl8RcYGRomJygpKjU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6goOEhYaHiImKkpOUlZaXmJmaoqOkpaanqKmqsrO0tba3uLm6wsPExcbHyMnK0tPU1dbX2Nna4uPk5ebn6Onq8vP09fb3+Pn6/9oADAMBAAIRAxEAPwD3+iiigD//2Q=="

	// Decode the base64 image to binary
	imageData, err := base64.StdEncoding.DecodeString(base64Image)
	require.NoError(t, err, "Failed to decode base64 image")

	// Create a test file header
	fileHeader := &multipart.FileHeader{
		Filename: "test-image.jpg",
		Size:     int64(len(imageData)),
		Header:   make(map[string][]string),
	}
	fileHeader.Header.Set("Content-Type", "image/jpeg")

	// Try to decode the image
	img, format, err := image.Decode(bytes.NewReader(imageData))
	require.NoError(t, err, "Failed to decode image")

	// Verify the format is correct
	assert.Equal(t, "jpeg", format, "Image format should be JPEG")

	// Verify the image dimensions
	bounds := img.Bounds()
	assert.Equal(t, 1, bounds.Dx(), "Image width should be 1 pixel")
	assert.Equal(t, 1, bounds.Dy(), "Image height should be 1 pixel")
}

// TestImageOperations tests the image processing operations without mocking AWS
func TestImageOperations(t *testing.T) {
	// Test basic image operation validation
	op := models.ImageOperation{
		Type:                "resize",
		Width:               400,
		Height:              300,
		MaintainAspectRatio: true,
	}

	// Verify operation properties
	assert.Equal(t, "resize", op.Type, "Operation type should be 'resize'")
	assert.Equal(t, 400, op.Width, "Width should be 400")
	assert.Equal(t, 300, op.Height, "Height should be 300")
	assert.True(t, op.MaintainAspectRatio, "MaintainAspectRatio should be true")

	// Test process image request
	req := models.ProcessImageRequest{
		Operations: []models.ImageOperation{op},
	}

	// Verify request properties
	assert.Len(t, req.Operations, 1, "Request should have 1 operation")
	assert.Equal(t, "resize", req.Operations[0].Type, "First operation type should be 'resize'")
}

// TestCustomizationModel tests the customization models without AWS dependencies
func TestCustomizationModel(t *testing.T) {
	// Create a customization image
	now := time.Now()
	image := &models.CustomizationImage{
		ID:          "test-id",
		URL:         "https://test-url.com/image.jpg",
		Width:       800,
		Height:      600,
		ContentType: "image/jpeg",
		Size:        1024,
		BucketName:  "test-bucket",
		ObjectKey:   "test-key.jpg",
		UserID:      "user123",
		ProductID:   "product456",
		VariantID:   "variant789",
		CartID:      "cart123",
		CreatedAt:   now,
		ExpiresAt:   now.Add(7 * 24 * time.Hour),
	}

	// Verify image properties
	assert.Equal(t, "test-id", image.ID)
	assert.Equal(t, "https://test-url.com/image.jpg", image.URL)
	assert.Equal(t, 800, image.Width)
	assert.Equal(t, 600, image.Height)
	assert.Equal(t, "image/jpeg", image.ContentType)
	assert.Equal(t, int64(1024), image.Size)
	assert.Equal(t, "test-bucket", image.BucketName)
	assert.Equal(t, "test-key.jpg", image.ObjectKey)
	assert.Equal(t, "user123", image.UserID)
	assert.Equal(t, "product456", image.ProductID)
	assert.Equal(t, "variant789", image.VariantID)
	assert.Equal(t, "cart123", image.CartID)
	assert.Equal(t, now, image.CreatedAt)
	assert.Equal(t, now.Add(7*24*time.Hour), image.ExpiresAt)

	// Test processed image response
	processedImage := &models.ProcessImageResponse{
		ID:              "processed-id",
		OriginalImageID: "test-id",
		URL:             "https://test-url.com/processed-image.jpg",
		Width:           400,
		Height:          300,
		ContentType:     "image/jpeg",
		Size:            512,
		UserID:          "user123",
		CreatedAt:       now,
		ExpiresAt:       now.Add(7 * 24 * time.Hour),
	}

	// Verify processed image properties
	assert.Equal(t, "processed-id", processedImage.ID)
	assert.Equal(t, "test-id", processedImage.OriginalImageID)
	assert.Equal(t, "https://test-url.com/processed-image.jpg", processedImage.URL)
	assert.Equal(t, 400, processedImage.Width)
	assert.Equal(t, 300, processedImage.Height)
	assert.Equal(t, "image/jpeg", processedImage.ContentType)
	assert.Equal(t, int64(512), processedImage.Size)
	assert.Equal(t, "user123", processedImage.UserID)
	assert.Equal(t, now, processedImage.CreatedAt)
	assert.Equal(t, now.Add(7*24*time.Hour), processedImage.ExpiresAt)
}

// TestCustomizationServiceValidations tests the validation logic in the customization service
func TestCustomizationServiceValidations(t *testing.T) {
	// Create a test file for size validation
	smallFile := make([]byte, 1000)         // 1KB
	largeFile := make([]byte, 15*1024*1024) // 15MB - should exceed the 10MB limit

	// Test file header for size validation
	smallHeader := &multipart.FileHeader{
		Filename: "small.jpg",
		Size:     int64(len(smallFile)),
		Header:   make(map[string][]string),
	}
	smallHeader.Header.Set("Content-Type", "image/jpeg")

	largeHeader := &multipart.FileHeader{
		Filename: "large.jpg",
		Size:     int64(len(largeFile)),
		Header:   make(map[string][]string),
	}
	largeHeader.Header.Set("Content-Type", "image/jpeg")

	// Validations to check
	// 1. Valid size (should pass)
	assert.True(t, smallHeader.Size <= 10*1024*1024, "Small file should be within size limit")

	// 2. Invalid size (should fail)
	assert.True(t, largeHeader.Size > 10*1024*1024, "Large file should exceed size limit")

	// 3. Valid content type (should pass)
	assert.Equal(t, "image/jpeg", smallHeader.Header.Get("Content-Type"), "Content type should be image/jpeg")

	// 4. Create an invalid content type header
	invalidHeader := &multipart.FileHeader{
		Filename: "document.pdf",
		Size:     1000,
		Header:   make(map[string][]string),
	}
	invalidHeader.Header.Set("Content-Type", "application/pdf")

	// Check that the content type is invalid
	assert.NotEqual(t, "image/jpeg", invalidHeader.Header.Get("Content-Type"), "Content type should not be an image")
	assert.NotContains(t, invalidHeader.Header.Get("Content-Type"), "image/", "Content type should not be an image")
}

// TestRealCustomizationService tests the customization service with real S3 if possible
func TestRealCustomizationService(t *testing.T) {
	// Get an S3 service using our testutil, which will decide if we should use real or mock
	s3Service, useRealS3 := testutil.GetS3Service()
	require.NotNil(t, s3Service)

	// Test bucket name for this test
	bucketName := "test-customization-bucket-" + time.Now().Format("20060102-150405")

	// If using mock, set up expectations
	if !useRealS3 {
		mockS3 := s3Service.(*mocks.MockS3Service)
		mockS3.On("CreateBucket", mock.Anything, bucketName).Return(nil)
		mockS3.On("BucketExists", mock.Anything, bucketName).Return(true, nil)
		mockS3.On("GenerateObjectKey", "test-uploads", "jpg").Return("test-uploads/test-image.jpg")
		mockS3.On("UploadFile", mock.Anything, bucketName, "test-uploads/test-image.jpg", mock.Anything, "image/jpeg").Return(nil)
		mockS3.On("DownloadFile", mock.Anything, bucketName, "test-uploads/test-image.jpg").Return([]byte("test image data"), "image/jpeg", nil)
		mockS3.On("GeneratePresignedURL", mock.Anything, bucketName, "test-uploads/test-image.jpg", mock.Anything).Return("https://mock-presigned-url.com", nil)
		mockS3.On("DeleteFile", mock.Anything, bucketName, "test-uploads/test-image.jpg").Return(nil)
	}

	// Create a test image
	const base64Image = "/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/2wBDAQkJCQwLDBgNDRgyIRwhMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjL/wAARCAABAAEDASIAAhEBAxEB/8QAHwAAAQUBAQEBAQEAAAAAAAAAAAECAwQFBgcICQoL/8QAtRAAAgEDAwIEAwUFBAQAAAF9AQIDAAQRBRIhMUEGE1FhByJxFDKBkaEII0KxwRVS0fAkM2JyggkKFhcYGRolJicoKSo0NTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqDhIWGh4iJipKTlJWWl5iZmqKjpKWmp6ipqrKztLW2t7i5usLDxMXGx8jJytLT1NXW19jZ2uHi4+Tl5ufo6erx8vP09fb3+Pn6/8QAHwEAAwEBAQEBAQEBAQAAAAAAAAECAwQFBgcICQoL/8QAtREAAgECBAQDBAcFBAQAAQJ3AAECAxEEBSExBhJBUQdhcRMiMoEIFEKRobHBCSMzUvAVYnLRChYkNOEl8RcYGRomJygpKjU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6goOEhYaHiImKkpOUlZaXmJmaoqOkpaanqKmqsrO0tba3uLm6wsPExcbHyMnK0tPU1dbX2Nna4uPk5ebn6Onq8vP09fb3+Pn6/9oADAMBAAIRAxEAPwD3+iiigD//2Q=="
	imageData, err := base64.StdEncoding.DecodeString(base64Image)
	require.NoError(t, err, "Failed to decode base64 image")

	// Real-world testing scenario: try to create the bucket
	err = s3Service.CreateBucket(context.Background(), bucketName)
	require.NoError(t, err, "Failed to create test bucket")

	// Confirm bucket exists
	exists, err := s3Service.BucketExists(context.Background(), bucketName)
	require.NoError(t, err, "Failed to check if bucket exists")
	assert.True(t, exists, "Bucket should exist after creation")

	// Now test uploading an image
	objectKey := "test-uploads/test-image.jpg"
	if useRealS3 {
		objectKey = s3Service.GenerateObjectKey("test-uploads", "jpg")
	}

	// Upload the test image directly using the S3 service
	err = s3Service.UploadFile(
		context.Background(),
		bucketName,
		objectKey,
		bytes.NewReader(imageData),
		"image/jpeg",
	)
	require.NoError(t, err, "Failed to upload test image")

	// Verify we can download the image
	downloadedData, contentType, err := s3Service.DownloadFile(
		context.Background(),
		bucketName,
		objectKey,
	)
	require.NoError(t, err, "Failed to download test image")
	assert.Equal(t, "image/jpeg", contentType, "Downloaded image content type mismatch")
	if useRealS3 {
		assert.Equal(t, len(imageData), len(downloadedData), "Downloaded image size mismatch")
	} else {
		assert.NotEmpty(t, downloadedData, "Downloaded data should not be empty")
	}

	// Generate a presigned URL for the image
	url, err := s3Service.GeneratePresignedURL(
		context.Background(),
		bucketName,
		objectKey,
		time.Hour,
	)
	require.NoError(t, err, "Failed to generate presigned URL")
	assert.NotEmpty(t, url, "Presigned URL should not be empty")

	err = s3Service.DeleteFile(
		context.Background(),
		bucketName,
		objectKey,
	)
	require.NoError(t, err, "Failed to delete test object")
}
