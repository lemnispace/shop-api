package unit

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/lemnispace/shop-api/internal/services"
	"github.com/lemnispace/shop-api/tests/mocks"
	"github.com/lemnispace/shop-api/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// S3Service is a wrapper interface for the services.S3Service interface
type S3Service = services.AWSS3Service

func TestS3Service(t *testing.T) {
	// Use our testutil to determine if we should use real S3 or mock
	s3Service, useRealS3 := testutil.GetS3Service()
	require.NotNil(t, s3Service)

	// Test bucket operations
	t.Run("BucketOperations", func(t *testing.T) {
		testBucketName := "test-bucket-" + time.Now().Format("20060102-150405")

		// If using mock, set up expectations
		if !useRealS3 {
			mockS3 := s3Service.(*mocks.MockS3Service)
			mockS3.On("CreateBucket", mock.Anything, testBucketName).Return(nil)
			mockS3.On("BucketExists", mock.Anything, testBucketName).Return(true, nil)
		}

		// Create bucket
		err := s3Service.CreateBucket(context.Background(), testBucketName)
		require.NoError(t, err)

		// Check if bucket exists
		exists, err := s3Service.BucketExists(context.Background(), testBucketName)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	// Test file operations
	t.Run("FileOperations", func(t *testing.T) {
		bucketName := "user-product-files"
		testData := "This is test data for S3 upload/download test"
		testContentType := "text/plain"
		objectKey := "test/mock-key.txt"

		// If using mock, set up expectations
		if !useRealS3 {
			mockS3 := s3Service.(*mocks.MockS3Service)
			// Clear previous expectations for these methods
			mockS3.ExpectedCalls = nil

			// Set up specific expectations for this test
			mockS3.On("GenerateObjectKey", "test", "txt").Return(objectKey)
			mockS3.On("UploadFile", mock.Anything, bucketName, objectKey, mock.Anything, testContentType).Return(nil)
			mockS3.On("DownloadFile", mock.Anything, bucketName, objectKey).Return([]byte(testData), testContentType, nil)
			mockS3.On("GeneratePresignedURL", mock.Anything, bucketName, objectKey, mock.Anything).Return("https://mock-presigned-url.com", nil)
			mockS3.On("DeleteFile", mock.Anything, bucketName, objectKey).Return(nil)

			// Add fallback expectations for other calls
			mockS3.On("BucketExists", mock.Anything, mock.Anything).Return(true, nil)
			mockS3.On("CreateBucket", mock.Anything, mock.Anything).Return(nil)
			mockS3.On("GeneratePresignedUploadURL", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("https://mock-upload-url.com", nil)
		}

		// Generate object key
		if useRealS3 {
			objectKey = s3Service.GenerateObjectKey("test", "txt")
		}

		// Upload file
		err := s3Service.UploadFile(
			context.Background(),
			bucketName,
			objectKey,
			strings.NewReader(testData),
			testContentType,
		)
		require.NoError(t, err)

		// Download file
		data, contentType, err := s3Service.DownloadFile(
			context.Background(),
			bucketName,
			objectKey,
		)
		require.NoError(t, err)
		assert.Equal(t, testData, string(data))
		assert.Equal(t, testContentType, contentType)

		// Generate presigned URL
		url, err := s3Service.GeneratePresignedURL(
			context.Background(),
			bucketName,
			objectKey,
			time.Hour,
		)
		require.NoError(t, err)
		assert.NotEmpty(t, url)

		// Delete file
		err = s3Service.DeleteFile(
			context.Background(),
			bucketName,
			objectKey,
		)
		require.NoError(t, err)
	})

	// Test presigned upload URL
	t.Run("PresignedUploadURL", func(t *testing.T) {
		bucketName := "user-product-files"
		testContentType := "image/jpeg"
		objectKey := "uploads/mock-image.jpg"

		// If using mock, set up expectations
		if !useRealS3 {
			mockS3 := s3Service.(*mocks.MockS3Service)
			// Clear previous expectations
			mockS3.ExpectedCalls = nil

			// Set up specific expectations for this test
			mockS3.On("GenerateObjectKey", "uploads", "jpg").Return(objectKey)
			mockS3.On("GeneratePresignedUploadURL", mock.Anything, bucketName, objectKey, testContentType, mock.Anything).Return("https://mock-upload-url.com", nil)

			// Add fallback expectations
			mockS3.On("BucketExists", mock.Anything, mock.Anything).Return(true, nil)
			mockS3.On("CreateBucket", mock.Anything, mock.Anything).Return(nil)
			mockS3.On("UploadFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockS3.On("DownloadFile", mock.Anything, mock.Anything, mock.Anything).Return([]byte("test data"), "text/plain", nil)
			mockS3.On("GeneratePresignedURL", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("https://mock-presigned-url.com", nil)
			mockS3.On("DeleteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		}

		// Generate object key
		if useRealS3 {
			objectKey = s3Service.GenerateObjectKey("uploads", "jpg")
		}

		// Generate presigned upload URL
		url, err := s3Service.GeneratePresignedUploadURL(
			context.Background(),
			bucketName,
			objectKey,
			testContentType,
			time.Hour,
		)
		require.NoError(t, err)
		assert.NotEmpty(t, url)
	})

	// Test object key generation
	t.Run("GenerateObjectKey", func(t *testing.T) {
		// If using mock, set up expectations
		if !useRealS3 {
			mockS3 := s3Service.(*mocks.MockS3Service)
			// Clear previous expectations
			mockS3.ExpectedCalls = nil

			// Set up specific expectations for this test
			mockS3.On("GenerateObjectKey", "test", "jpg").Return("test/mock-key.jpg")
			mockS3.On("GenerateObjectKey", "", "png").Return("mock-key.png")
			mockS3.On("GenerateObjectKey", "folder", "").Return("folder/mock-key")

			// Add fallback expectations
			mockS3.On("BucketExists", mock.Anything, mock.Anything).Return(true, nil)
			mockS3.On("CreateBucket", mock.Anything, mock.Anything).Return(nil)
			mockS3.On("UploadFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockS3.On("DownloadFile", mock.Anything, mock.Anything, mock.Anything).Return([]byte("test data"), "text/plain", nil)
			mockS3.On("GeneratePresignedURL", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("https://mock-presigned-url.com", nil)
			mockS3.On("DeleteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockS3.On("GeneratePresignedUploadURL", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("https://mock-upload-url.com", nil)
		}

		// With prefix and extension
		key1 := s3Service.GenerateObjectKey("test", "jpg")
		if useRealS3 {
			assert.Contains(t, key1, "test/")
			assert.Contains(t, key1, ".jpg")
		} else {
			assert.Equal(t, "test/mock-key.jpg", key1)
		}

		// Without prefix
		key2 := s3Service.GenerateObjectKey("", "png")
		if useRealS3 {
			assert.NotContains(t, key2, "/")
			assert.Contains(t, key2, ".png")
		} else {
			assert.Equal(t, "mock-key.png", key2)
		}

		// Without extension
		key3 := s3Service.GenerateObjectKey("folder", "")
		if useRealS3 {
			assert.Contains(t, key3, "folder/")
			assert.NotContains(t, key3, ".")
		} else {
			assert.Equal(t, "folder/mock-key", key3)
		}
	})
}
