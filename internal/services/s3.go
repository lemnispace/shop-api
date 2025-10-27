package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"github.com/lemnispace/shop-api/internal/utils"
)

var (
	ErrS3ClientNotInitialized = errors.New("S3 client not initialized")
	ErrInvalidBucketName      = errors.New("invalid bucket name")
	ErrInvalidObjectKey       = errors.New("invalid object key")
	ErrObjectNotFound         = errors.New("object not found")
	ErrUploadFailed           = errors.New("failed to upload file")
	ErrDownloadFailed         = errors.New("failed to download file")
)

// S3Service defines the interface for S3 operations
type S3Service interface {
	// File operations
	UploadFile(ctx context.Context, bucketName, objectKey string, data io.Reader, contentType string) error
	DownloadFile(ctx context.Context, bucketName, objectKey string) ([]byte, string, error)
	DeleteFile(ctx context.Context, bucketName, objectKey string) error

	// Presigned URLs
	GeneratePresignedURL(ctx context.Context, bucketName, objectKey string, expires time.Duration) (string, error)
	GeneratePresignedUploadURL(ctx context.Context, bucketName, objectKey, contentType string, expires time.Duration) (string, error)

	// Bucket operations
	CreateBucket(ctx context.Context, bucketName string) error
	BucketExists(ctx context.Context, bucketName string) (bool, error)

	// Utility methods
	GenerateObjectKey(prefix, extension string) string
}

// AWSS3Service is an implementation of S3Service using AWS SDK
type AWSS3Service struct {
	client         *s3.Client
	endpoint       string
	region         string
	forcePathStyle bool
}

// NewS3Service creates a new S3 service with the provided configuration
func NewS3Service() (*AWSS3Service, error) {
	// Check for local environment variables
	endpoint := os.Getenv("S3_ENDPOINT")
	region := os.Getenv("S3_REGION")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")

	// Set defaults if not provided
	if region == "" {
		region = "us-east-1"
	}

	var client *s3.Client

	// Check if we're using a local endpoint (like MinIO)
	if endpoint != "" {
		// For local development with MinIO
		utils.DebugLog("Initializing S3 client with local endpoint: %s", endpoint)

		// Custom resolver that uses the provided endpoint
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               endpoint,
				HostnameImmutable: true,
				SigningRegion:     region,
			}, nil
		})

		// Create a custom configuration
		cfg, err := config.LoadDefaultConfig(context.Background(),
			config.WithRegion(region),
			config.WithEndpointResolverWithOptions(customResolver),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load S3 config: %w", err)
		}

		// Create the S3 client with the custom configuration
		client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.UsePathStyle = true // Important for MinIO compatibility
		})

		utils.DebugLog("Successfully initialized S3 client with local endpoint")
	} else {
		// For production AWS
		utils.DebugLog("Initializing S3 client with AWS endpoint")

		// Load the default AWS configuration
		cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
		if err != nil {
			return nil, fmt.Errorf("failed to load S3 config: %w", err)
		}

		// Create the S3 client with the default configuration
		client = s3.NewFromConfig(cfg)

		utils.DebugLog("Successfully initialized S3 client with AWS endpoint")
	}

	// Determine if we need force path style (required for MinIO)
	forcePathStyle := false
	if endpoint != "" {
		forcePathStyle = true
	}

	return &AWSS3Service{
		client:         client,
		endpoint:       endpoint,
		region:         region,
		forcePathStyle: forcePathStyle,
	}, nil
}

// UploadFile uploads a file to S3
func (s *AWSS3Service) UploadFile(ctx context.Context, bucketName, objectKey string, data io.Reader, contentType string) error {
	if s.client == nil {
		return ErrS3ClientNotInitialized
	}

	if bucketName == "" {
		return ErrInvalidBucketName
	}

	if objectKey == "" {
		return ErrInvalidObjectKey
	}

	utils.DebugLog("Uploading file to S3 - Bucket: %s, Key: %s, Content-Type: %s", bucketName, objectKey, contentType)

	// Read the data into a buffer to get the size
	buf := new(bytes.Buffer)
	size, err := io.Copy(buf, data)
	if err != nil {
		utils.ErrorLog("Failed to read data for upload: %v", err)
		return fmt.Errorf("failed to read data: %w", err)
	}

	// Set up upload parameters
	params := &s3.PutObjectInput{
		Bucket:        aws.String(bucketName),
		Key:           aws.String(objectKey),
		Body:          bytes.NewReader(buf.Bytes()),
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
	}

	// Perform the upload
	_, err = s.client.PutObject(ctx, params)
	if err != nil {
		utils.ErrorLog("Failed to upload file to S3: %v", err)
		return fmt.Errorf("%w: %v", ErrUploadFailed, err)
	}

	utils.DebugLog("Successfully uploaded file to S3 - Bucket: %s, Key: %s", bucketName, objectKey)
	return nil
}

// DownloadFile downloads a file from S3
func (s *AWSS3Service) DownloadFile(ctx context.Context, bucketName, objectKey string) ([]byte, string, error) {
	if s.client == nil {
		return nil, "", ErrS3ClientNotInitialized
	}

	if bucketName == "" {
		return nil, "", ErrInvalidBucketName
	}

	if objectKey == "" {
		return nil, "", ErrInvalidObjectKey
	}

	utils.DebugLog("Downloading file from S3 - Bucket: %s, Key: %s", bucketName, objectKey)

	// Set up download parameters
	params := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}

	// Perform the download
	resp, err := s.client.GetObject(ctx, params)
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			utils.ErrorLog("Object not found in S3: %s/%s", bucketName, objectKey)
			return nil, "", ErrObjectNotFound
		}
		utils.ErrorLog("Failed to download file from S3: %v", err)
		return nil, "", fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}

	// Read the data
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.ErrorLog("Failed to read S3 response: %v", err)
		return nil, "", fmt.Errorf("failed to read download data: %w", err)
	}

	// Get content type
	contentType := ""
	if resp.ContentType != nil {
		contentType = *resp.ContentType
	}

	utils.DebugLog("Successfully downloaded file from S3 - Bucket: %s, Key: %s, Size: %d bytes", bucketName, objectKey, len(data))
	return data, contentType, nil
}

// DeleteFile deletes a file from S3
func (s *AWSS3Service) DeleteFile(ctx context.Context, bucketName, objectKey string) error {
	if s.client == nil {
		return ErrS3ClientNotInitialized
	}

	if bucketName == "" {
		return ErrInvalidBucketName
	}

	if objectKey == "" {
		return ErrInvalidObjectKey
	}

	utils.DebugLog("Deleting file from S3 - Bucket: %s, Key: %s", bucketName, objectKey)

	// Set up delete parameters
	params := &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}

	// Perform the delete
	_, err := s.client.DeleteObject(ctx, params)
	if err != nil {
		utils.ErrorLog("Failed to delete file from S3: %v", err)
		return fmt.Errorf("failed to delete object: %w", err)
	}

	utils.DebugLog("Successfully deleted file from S3 - Bucket: %s, Key: %s", bucketName, objectKey)
	return nil
}

// GeneratePresignedURL generates a presigned URL for downloading a file
func (s *AWSS3Service) GeneratePresignedURL(ctx context.Context, bucketName, objectKey string, expires time.Duration) (string, error) {
	if s.client == nil {
		return "", ErrS3ClientNotInitialized
	}

	if bucketName == "" {
		return "", ErrInvalidBucketName
	}

	if objectKey == "" {
		return "", ErrInvalidObjectKey
	}

	utils.DebugLog("Generating presigned URL for S3 object - Bucket: %s, Key: %s, Expires: %v", bucketName, objectKey, expires)

	// TODO(security): Generate real presigned URLs even when using a custom/MinIO endpoint so private
	// buckets stay private and expirations are enforced instead of returning unsigned object URLs.
	// For local MinIO setup, we need to generate the URL directly
	if s.endpoint != "" && s.forcePathStyle {
		// Check if object exists first
		_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(objectKey),
		})
		if err != nil {
			utils.ErrorLog("Object does not exist in S3: %v", err)
			return "", ErrObjectNotFound
		}

		// For MinIO, construct the URL manually
		u, err := url.Parse(s.endpoint)
		if err != nil {
			return "", fmt.Errorf("failed to parse endpoint URL: %w", err)
		}

		u.Path = fmt.Sprintf("/%s/%s", bucketName, objectKey)
		preSignedURL := u.String()

		utils.DebugLog("Generated presigned URL for MinIO: %s", preSignedURL)
		return preSignedURL, nil
	}

	// For AWS S3, use the presigner
	presignClient := s3.NewPresignClient(s.client)
	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expires
	})

	if err != nil {
		utils.ErrorLog("Failed to generate presigned URL: %v", err)
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	utils.DebugLog("Successfully generated presigned URL for S3 object: %s", request.URL)
	return request.URL, nil
}

// GeneratePresignedUploadURL generates a presigned URL for uploading a file
func (s *AWSS3Service) GeneratePresignedUploadURL(ctx context.Context, bucketName, objectKey, contentType string, expires time.Duration) (string, error) {
	if s.client == nil {
		return "", ErrS3ClientNotInitialized
	}

	if bucketName == "" {
		return "", ErrInvalidBucketName
	}

	if objectKey == "" {
		return "", ErrInvalidObjectKey
	}

	utils.DebugLog("Generating presigned upload URL for S3 - Bucket: %s, Key: %s, ContentType: %s, Expires: %v",
		bucketName, objectKey, contentType, expires)

	// For local MinIO setup with path style, we need to handle differently
	if s.endpoint != "" && s.forcePathStyle {
		// For MinIO, construct the URL manually
		u, err := url.Parse(s.endpoint)
		if err != nil {
			return "", fmt.Errorf("failed to parse endpoint URL: %w", err)
		}

		u.Path = fmt.Sprintf("/%s/%s", bucketName, objectKey)

		// Add query parameters for the upload
		query := u.Query()
		query.Set("X-Amz-ContentType", contentType)
		u.RawQuery = query.Encode()

		preSignedURL := u.String()

		utils.DebugLog("Generated presigned upload URL for MinIO: %s", preSignedURL)
		return preSignedURL, nil
	}

	// For AWS S3, use the presigner
	presignClient := s3.NewPresignClient(s.client)
	request, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(objectKey),
		ContentType: aws.String(contentType),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expires
	})

	if err != nil {
		utils.ErrorLog("Failed to generate presigned upload URL: %v", err)
		return "", fmt.Errorf("failed to generate presigned upload URL: %w", err)
	}

	utils.DebugLog("Successfully generated presigned upload URL for S3: %s", request.URL)
	return request.URL, nil
}

// CreateBucket creates a new S3 bucket
func (s *AWSS3Service) CreateBucket(ctx context.Context, bucketName string) error {
	if s.client == nil {
		return ErrS3ClientNotInitialized
	}

	if bucketName == "" {
		return ErrInvalidBucketName
	}

	utils.DebugLog("Creating S3 bucket: %s", bucketName)

	// Set up create bucket parameters
	params := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	// For non-us-east-1 regions, we need to specify the location constraint
	if s.region != "us-east-1" {
		params.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(s.region),
		}
	}

	// Create the bucket
	_, err := s.client.CreateBucket(ctx, params)
	if err != nil {
		var bae *types.BucketAlreadyExists
		var baoby *types.BucketAlreadyOwnedByYou

		if errors.As(err, &bae) || errors.As(err, &baoby) {
			utils.DebugLog("Bucket %s already exists", bucketName)
			return nil
		}

		utils.ErrorLog("Failed to create S3 bucket: %v", err)
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	utils.DebugLog("Successfully created S3 bucket: %s", bucketName)
	return nil
}

// BucketExists checks if a bucket exists
func (s *AWSS3Service) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	if s.client == nil {
		return false, ErrS3ClientNotInitialized
	}

	if bucketName == "" {
		return false, ErrInvalidBucketName
	}

	utils.DebugLog("Checking if S3 bucket exists: %s", bucketName)

	// Try to get the bucket location to check if it exists
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})

	if err != nil {
		var nsk *types.NoSuchBucket
		if errors.As(err, &nsk) {
			utils.DebugLog("Bucket %s does not exist", bucketName)
			return false, nil
		}

		utils.ErrorLog("Error checking bucket existence: %v", err)
		return false, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	utils.DebugLog("Bucket %s exists", bucketName)
	return true, nil
}

// GenerateObjectKey generates a unique object key for S3 storage
func (s *AWSS3Service) GenerateObjectKey(prefix, extension string) string {
	// Clean prefix and ensure it doesn't start with a slash
	prefix = strings.Trim(strings.TrimSpace(prefix), "/")
	if prefix != "" {
		prefix = prefix + "/"
	}

	// Clean extension and ensure it starts with a dot
	extension = strings.TrimSpace(extension)
	if extension != "" && !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}

	// Generate a unique filename using UUID
	filename := uuid.New().String() + extension

	// Return the complete object key
	return prefix + filename
}
