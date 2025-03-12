package mocks

import (
	"context"
	"io"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockS3Service provides a mock implementation of the S3 service interface
type MockS3Service struct {
	mock.Mock
}

func (m *MockS3Service) UploadFile(ctx context.Context, bucketName, objectKey string, fileContent io.Reader, contentType string) error {
	args := m.Called(ctx, bucketName, objectKey, fileContent, contentType)
	return args.Error(0)
}

func (m *MockS3Service) DownloadFile(ctx context.Context, bucketName, objectKey string) ([]byte, string, error) {
	args := m.Called(ctx, bucketName, objectKey)
	return args.Get(0).([]byte), args.String(1), args.Error(2)
}

func (m *MockS3Service) DeleteFile(ctx context.Context, bucketName, objectKey string) error {
	args := m.Called(ctx, bucketName, objectKey)
	return args.Error(0)
}

func (m *MockS3Service) GeneratePresignedURL(ctx context.Context, bucketName, objectKey string, expiration time.Duration) (string, error) {
	args := m.Called(ctx, bucketName, objectKey, expiration)
	return args.String(0), args.Error(1)
}

func (m *MockS3Service) GenerateObjectKey(prefix, extension string) string {
	args := m.Called(prefix, extension)
	return args.String(0)
}

func (m *MockS3Service) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	args := m.Called(ctx, bucketName)
	return args.Bool(0), args.Error(1)
}

func (m *MockS3Service) CreateBucket(ctx context.Context, bucketName string) error {
	args := m.Called(ctx, bucketName)
	return args.Error(0)
}

func (m *MockS3Service) GeneratePresignedUploadURL(ctx context.Context, bucketName, objectKey, contentType string, expiration time.Duration) (string, error) {
	args := m.Called(ctx, bucketName, objectKey, contentType, expiration)
	return args.String(0), args.Error(1)
}

// NewMockS3Service creates a preconfigured mock S3 service with default success responses
func NewMockS3Service() *MockS3Service {
	mockS3 := new(MockS3Service)

	// Configure the mock with default expectations
	mockS3.On("CreateBucket", mock.Anything, mock.Anything).Return(nil)
	mockS3.On("BucketExists", mock.Anything, mock.Anything).Return(true, nil)
	mockS3.On("GenerateObjectKey", mock.Anything, mock.Anything).Return("test-uploads/test-image.jpg")
	mockS3.On("UploadFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockS3.On("DownloadFile", mock.Anything, mock.Anything, mock.Anything).Return([]byte("test image data"), "image/jpeg", nil)
	mockS3.On("GeneratePresignedURL", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("https://mock-presigned-url.com", nil)
	mockS3.On("DeleteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockS3.On("GeneratePresignedUploadURL", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("https://mock-upload-url.com", nil)

	return mockS3
}

// NewSpecificMockS3Service creates a mock S3 service for a specific bucket and object key
func NewSpecificMockS3Service(bucketName, objectKey string, fileData []byte, contentType string) *MockS3Service {
	mockS3 := new(MockS3Service)

	// Configure for the specific bucket and object key
	mockS3.On("BucketExists", mock.Anything, bucketName).Return(true, nil)
	mockS3.On("BucketExists", mock.Anything, mock.Anything).Return(false, nil)

	mockS3.On("CreateBucket", mock.Anything, bucketName).Return(nil)
	mockS3.On("CreateBucket", mock.Anything, mock.Anything).Return(nil)

	mockS3.On("GenerateObjectKey", mock.Anything, mock.Anything).Return(objectKey)

	mockS3.On("UploadFile", mock.Anything, bucketName, objectKey, mock.Anything, mock.Anything).Return(nil)
	mockS3.On("UploadFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	mockS3.On("DownloadFile", mock.Anything, bucketName, objectKey).Return(fileData, contentType, nil)
	mockS3.On("DownloadFile", mock.Anything, mock.Anything, mock.Anything).Return([]byte("default test data"), "application/octet-stream", nil)

	mockS3.On("GeneratePresignedURL", mock.Anything, bucketName, objectKey, mock.Anything).Return("https://mock-presigned-url.com/"+bucketName+"/"+objectKey, nil)
	mockS3.On("GeneratePresignedURL", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("https://mock-presigned-url.com/default", nil)

	mockS3.On("DeleteFile", mock.Anything, bucketName, objectKey).Return(nil)
	mockS3.On("DeleteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	mockS3.On("GeneratePresignedUploadURL", mock.Anything, bucketName, objectKey, contentType, mock.Anything).Return("https://mock-upload-url.com/"+bucketName+"/"+objectKey, nil)
	mockS3.On("GeneratePresignedUploadURL", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("https://mock-upload-url.com/default", nil)

	return mockS3
}
