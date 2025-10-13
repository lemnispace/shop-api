package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"mime/multipart"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/internal/utils"
)

const (
	// Image storage configuration
	DefaultImageBucket   = "user-product-files"
	CustomizationPrefix  = "customizations/"
	ProcessedPrefix      = "customizations/processed/"
	DefaultImageLifetime = 24 * time.Hour * 7 // 7 days
)

var (
	ErrInvalidImage      = errors.New("invalid image format")
	ErrImageTooLarge     = errors.New("image too large")
	ErrOperationNotFound = errors.New("operation not found")
	ErrInvalidOperation  = errors.New("invalid operation")
)

// CustomizationService defines the interface for customization operations
type CustomizationService interface {
	UploadImage(ctx context.Context, file multipart.File, fileHeader *multipart.FileHeader, userID, cartID, productID, variantID string) (*models.CustomizationImage, error)
	GetImage(ctx context.Context, imageID string) (*models.CustomizationImage, error)
	ProcessImage(ctx context.Context, imageID string, request models.ProcessImageRequest) (*models.ProcessImageResponse, error)
	DeleteImage(ctx context.Context, imageID string) error
	GetImagesByUserAndProduct(ctx context.Context, userID, productID, variantID string) ([]*models.CustomizationImage, error)
	LinkImageToCartItem(ctx context.Context, imageID, cartID, cartItemID string) error
}

// DynamoDBCustomizationService is an implementation of CustomizationService using DynamoDB and S3
type DynamoDBCustomizationService struct {
	db         *dynamodb.Client
	s3Service  S3Service
	tableName  string
	bucketName string
}

// NewCustomizationService creates a new customization service
func NewCustomizationService(db *dynamodb.Client, s3Service S3Service, tableName string) *DynamoDBCustomizationService {
	bucketName := os.Getenv("S3_USER_FILES_BUCKET")
	if bucketName == "" {
		utils.ErrorLog("S3_USER_FILES_BUCKET environment variable not set, using default bucket name")
		bucketName = DefaultImageBucket
	} else {
		utils.DebugLog("Using S3 bucket from environment: %s", bucketName)
	}

	return &DynamoDBCustomizationService{
		db:         db,
		s3Service:  s3Service,
		tableName:  tableName,
		bucketName: bucketName,
	}
}

// UploadImage uploads a customization image to S3 and stores metadata in DynamoDB
func (s *DynamoDBCustomizationService) UploadImage(
	ctx context.Context,
	file multipart.File,
	fileHeader *multipart.FileHeader,
	userID, cartID, productID, variantID string,
) (*models.CustomizationImage, error) {
	utils.DebugLog("Uploading customization image - Size: %d bytes, ContentType: %s", fileHeader.Size, fileHeader.Header.Get("Content-Type"))

	// Validate file size (10MB max)
	if fileHeader.Size > 10*1024*1024 {
		return nil, ErrImageTooLarge
	}

	// Validate content type
	contentType := fileHeader.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return nil, ErrInvalidImage
	}

	// Read the image data to get dimensions
	imgData, err := io.ReadAll(file)
	if err != nil {
		utils.ErrorLog("Failed to read image data: %v", err)
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	// Reset file position for S3 upload
	file.Seek(0, io.SeekStart)

	// Parse the image to get dimensions
	img, imgFormat, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		utils.ErrorLog("Failed to decode image: %v", err)
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Get image dimensions
	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y

	// Generate a unique ID for the image
	imageID := uuid.New().String()

	// Generate S3 object key
	extension := ""
	switch imgFormat {
	case "jpeg":
		extension = "jpg"
	case "png":
		extension = "png"
	case "gif":
		extension = "gif"
	case "webp":
		extension = "webp"
	default:
		extension = strings.TrimPrefix(contentType, "image/")
	}

	// Include userID in the object key path for better organization
	prefixWithUser := CustomizationPrefix
	if userID != "" {
		prefixWithUser = fmt.Sprintf("%s%s/", CustomizationPrefix, userID)
	}

	objectKey := s.s3Service.GenerateObjectKey(prefixWithUser, extension)

	// Upload the image to S3
	err = s.s3Service.UploadFile(ctx, s.bucketName, objectKey, bytes.NewReader(imgData), contentType)
	if err != nil {
		utils.ErrorLog("Failed to upload image to S3: %v", err)
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	// Generate expiration time (7 days from now)
	createdAt := time.Now()
	expiresAt := createdAt.Add(DefaultImageLifetime)

	// Generate a presigned URL for the image
	url, err := s.s3Service.GeneratePresignedURL(ctx, s.bucketName, objectKey, DefaultImageLifetime)
	if err != nil {
		utils.ErrorLog("Failed to generate presigned URL: %v", err)
		return nil, fmt.Errorf("failed to generate URL: %w", err)
	}

	// Create the image record
	image := &models.CustomizationImage{
		ID:          imageID,
		URL:         url,
		Width:       width,
		Height:      height,
		ContentType: contentType,
		Size:        fileHeader.Size,
		UserID:      userID,
		CartID:      cartID,
		ProductID:   productID,
		VariantID:   variantID,
		CreatedAt:   createdAt,
		ExpiresAt:   expiresAt,
	}

	// Store image metadata in DynamoDB
	item := map[string]interface{}{
		"PK":          fmt.Sprintf("CUSTOMIZATION#%s", imageID),
		"SK":          "METADATA",
		"ID":          imageID,
		"URL":         url,
		"Width":       width,
		"Height":      height,
		"ContentType": contentType,
		"Size":        fileHeader.Size,
		"BucketName":  s.bucketName,
		"ObjectKey":   objectKey,
		"CreatedAt":   createdAt.Format(time.RFC3339),
		"ExpiresAt":   expiresAt.Format(time.RFC3339),
		"Type":        "CustomizationImage",
	}

	// Add optional fields if they exist
	if userID != "" {
		item["UserID"] = userID
		// Add an additional GSI key for querying by user
		item["GSI1PK"] = fmt.Sprintf("USER#%s", userID)
		item["GSI1SK"] = fmt.Sprintf("CUSTOMIZATION#%s", imageID)
	}
	if cartID != "" {
		item["CartID"] = cartID
	}
	if productID != "" {
		item["ProductID"] = productID

		// If we have both product and variant, add another GSI key for querying by product and variant
		if variantID != "" {
			item["GSI2PK"] = fmt.Sprintf("PRODUCT#%s", productID)
			item["GSI2SK"] = fmt.Sprintf("VARIANT#%s#CUSTOMIZATION#%s", variantID, imageID)
		}
	}
	if variantID != "" {
		item["VariantID"] = variantID
	}

	// Store the item in DynamoDB
	err = PutItem(ctx, s.db, s.tableName, item)
	if err != nil {
		utils.ErrorLog("Failed to store image metadata in DynamoDB: %v", err)
		// Try to clean up the S3 object if we can't store the metadata
		_ = s.s3Service.DeleteFile(ctx, s.bucketName, objectKey)
		return nil, fmt.Errorf("failed to store image metadata: %w", err)
	}

	utils.DebugLog("Successfully uploaded customization image - ID: %s, Size: %d, Dimensions: %dx%d",
		imageID, fileHeader.Size, width, height)

	return image, nil
}

// GetImage retrieves a customization image by ID
func (s *DynamoDBCustomizationService) GetImage(ctx context.Context, imageID string) (*models.CustomizationImage, error) {
	utils.DebugLog("Getting customization image - ID: %s", imageID)

	// Retrieve image metadata from DynamoDB
	key := map[string]interface{}{
		"PK": fmt.Sprintf("CUSTOMIZATION#%s", imageID),
		"SK": "METADATA",
	}

	item, err := GetItem(ctx, s.db, s.tableName, key)
	if err != nil {
		utils.ErrorLog("Failed to get image metadata from DynamoDB: %v", err)
		return nil, fmt.Errorf("failed to get image metadata: %w", err)
	}

	if item == nil {
		utils.ErrorLog("Image not found - ID: %s", imageID)
		return nil, ErrObjectNotFound
	}

	// Parse the item into a CustomizationImage
	image := &models.CustomizationImage{
		ID:          imageID,
		URL:         item["URL"].(string),
		Width:       int(item["Width"].(float64)),
		Height:      int(item["Height"].(float64)),
		ContentType: item["ContentType"].(string),
		Size:        int64(item["Size"].(float64)),
		CreatedAt:   ParseTime(item["CreatedAt"].(string)),
		ExpiresAt:   ParseTime(item["ExpiresAt"].(string)),
	}

	// Optional fields
	if userID, ok := item["UserID"].(string); ok && userID != "" {
		image.UserID = userID
	}
	if cartID, ok := item["CartID"].(string); ok && cartID != "" {
		image.CartID = cartID
	}
	if productID, ok := item["ProductID"].(string); ok && productID != "" {
		image.ProductID = productID
	}
	if variantID, ok := item["VariantID"].(string); ok && variantID != "" {
		image.VariantID = variantID
	}

	// Get bucket name and object key from DynamoDB for generating URL
	bucketName, _ := item["BucketName"].(string)
	objectKey, _ := item["ObjectKey"].(string)

	// Generate a fresh presigned URL if we have the bucket and key
	if bucketName != "" && objectKey != "" {
		url, err := s.s3Service.GeneratePresignedURL(ctx, bucketName, objectKey, time.Until(image.ExpiresAt))
		if err != nil {
			utils.ErrorLog("Failed to generate presigned URL: %v", err)
			return nil, fmt.Errorf("failed to generate URL: %w", err)
		}
		image.URL = url
	}

	utils.DebugLog("Successfully retrieved customization image - ID: %s", imageID)
	return image, nil
}

// ProcessImage processes an uploaded image according to the requested operations
func (s *DynamoDBCustomizationService) ProcessImage(
	ctx context.Context,
	imageID string,
	request models.ProcessImageRequest,
) (*models.ProcessImageResponse, error) {
	utils.DebugLog("Processing image - ID: %s, Operations: %d", imageID, len(request.Operations))

	// Get the original image metadata from DynamoDB
	key := map[string]interface{}{
		"PK": fmt.Sprintf("CUSTOMIZATION#%s", imageID),
		"SK": "METADATA",
	}

	item, err := GetItem(ctx, s.db, s.tableName, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get original image metadata: %w", err)
	}

	if item == nil {
		return nil, ErrObjectNotFound
	}

	// Get the original image using GetImage to get the full model
	originalImage, err := s.GetImage(ctx, imageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original image: %w", err)
	}

	// Get bucket name and object key from DynamoDB for downloading
	bucketName, _ := item["BucketName"].(string)
	objectKey, _ := item["ObjectKey"].(string)

	if bucketName == "" || objectKey == "" {
		return nil, fmt.Errorf("bucket name or object key not found in image metadata")
	}

	// Download the image from S3
	imageData, contentType, err := s.s3Service.DownloadFile(ctx, bucketName, objectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}

	// Note: Image processing operations (resize, crop, filter, etc.) are not yet implemented.
	// Currently, this function validates the request and stores the original image as "processed".
	// Future implementation should integrate with an image processing library like imaging or vips.

	// Generate a new ID for the processed image
	processedID := uuid.New().String()

	// Generate a new object key for the processed image
	extension := strings.Split(contentType, "/")[1]

	// Include userID in the processed image path if available
	processedPrefix := ProcessedPrefix
	if originalImage.UserID != "" {
		processedPrefix = fmt.Sprintf("%s%s/", ProcessedPrefix, originalImage.UserID)
	}

	processedKey := s.s3Service.GenerateObjectKey(processedPrefix, extension)

	// Upload the processed image (using the original image for now)
	err = s.s3Service.UploadFile(ctx, s.bucketName, processedKey, bytes.NewReader(imageData), contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to upload processed image: %w", err)
	}

	// Generate expiration time (7 days from now)
	createdAt := time.Now()
	expiresAt := createdAt.Add(DefaultImageLifetime)

	// Generate a presigned URL for the processed image
	url, err := s.s3Service.GeneratePresignedURL(ctx, s.bucketName, processedKey, DefaultImageLifetime)
	if err != nil {
		return nil, fmt.Errorf("failed to generate URL for processed image: %w", err)
	}

	// Create the response
	response := &models.ProcessImageResponse{
		ID:              processedID,
		OriginalImageID: originalImage.ID,
		URL:             url,
		Width:           originalImage.Width,  // Would be updated with actual dimensions after processing
		Height:          originalImage.Height, // Would be updated with actual dimensions after processing
		ContentType:     contentType,
		Size:            int64(len(imageData)),
		UserID:          originalImage.UserID, // Carry over the userID
		CreatedAt:       createdAt,
		ExpiresAt:       expiresAt,
	}

	// Store processed image metadata in DynamoDB
	processedItem := map[string]interface{}{
		"PK":              fmt.Sprintf("CUSTOMIZATION#%s", processedID),
		"SK":              "METADATA",
		"ID":              processedID,
		"OriginalImageID": originalImage.ID,
		"URL":             url,
		"Width":           originalImage.Width,
		"Height":          originalImage.Height,
		"ContentType":     contentType,
		"Size":            int64(len(imageData)),
		"BucketName":      s.bucketName,
		"ObjectKey":       processedKey,
		"Type":            "ProcessedCustomizationImage",
		"CreatedAt":       createdAt.Format(time.RFC3339),
		"ExpiresAt":       expiresAt.Format(time.RFC3339),
	}

	// Add user ID from original image if available
	if originalImage.UserID != "" {
		processedItem["UserID"] = originalImage.UserID

		// Add GSI1 for querying by user
		processedItem["GSI1PK"] = fmt.Sprintf("USER#%s", originalImage.UserID)
		processedItem["GSI1SK"] = fmt.Sprintf("PROCESSED#%s", processedID)
	}

	// Add GSI2 for querying by product/variant if available
	if originalImage.ProductID != "" && originalImage.VariantID != "" {
		processedItem["ProductID"] = originalImage.ProductID
		processedItem["VariantID"] = originalImage.VariantID
		processedItem["GSI2PK"] = fmt.Sprintf("PRODUCT#%s", originalImage.ProductID)
		processedItem["GSI2SK"] = fmt.Sprintf("VARIANT#%s#PROCESSED#%s", originalImage.VariantID, processedID)
	}

	// Store the metadata in DynamoDB
	err = PutItem(ctx, s.db, s.tableName, processedItem)
	if err != nil {
		utils.ErrorLog("Failed to store processed image metadata: %v", err)
		// Try to clean up the S3 object if we can't store the metadata
		_ = s.s3Service.DeleteFile(ctx, s.bucketName, processedKey)
		return nil, fmt.Errorf("failed to store processed image metadata: %w", err)
	}

	utils.DebugLog("Successfully processed image - Original ID: %s, Processed ID: %s", imageID, processedID)

	return response, nil
}

// DeleteImage deletes a customization image from S3 and its metadata from DynamoDB
func (s *DynamoDBCustomizationService) DeleteImage(ctx context.Context, imageID string) error {
	utils.DebugLog("Deleting customization image - ID: %s", imageID)

	// Get the image metadata from DynamoDB (not using GetImage to avoid circular logic)
	key := map[string]interface{}{
		"PK": fmt.Sprintf("CUSTOMIZATION#%s", imageID),
		"SK": "METADATA",
	}

	item, err := GetItem(ctx, s.db, s.tableName, key)
	if err != nil {
		utils.ErrorLog("Failed to get image metadata for deletion: %v", err)
		return fmt.Errorf("failed to get image metadata: %w", err)
	}

	if item == nil {
		utils.ErrorLog("Image not found - ID: %s", imageID)
		return ErrObjectNotFound
	}

	// Get bucket name and object key from DynamoDB
	bucketName, _ := item["BucketName"].(string)
	objectKey, _ := item["ObjectKey"].(string)

	// Delete the image from S3 if we have the bucket and key
	if bucketName != "" && objectKey != "" {
		err = s.s3Service.DeleteFile(ctx, bucketName, objectKey)
		if err != nil {
			utils.ErrorLog("Failed to delete image from S3: %v", err)
			return fmt.Errorf("failed to delete image from S3: %w", err)
		}
	}

	// Delete the metadata from DynamoDB (reuse the same key)
	err = DeleteItem(ctx, s.db, s.tableName, key)
	if err != nil {
		utils.ErrorLog("Failed to delete image metadata from DynamoDB: %v", err)
		return fmt.Errorf("failed to delete image metadata: %w", err)
	}

	utils.DebugLog("Successfully deleted customization image - ID: %s", imageID)
	return nil
}

// ParseTime parses a time string in RFC3339 format
func ParseTime(timeStr string) time.Time {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return time.Time{}
	}
	return t
}

// PutItem adds or updates an item in DynamoDB
func PutItem(ctx context.Context, db *dynamodb.Client, tableName string, item map[string]interface{}) error {
	// Convert the item to DynamoDB attribute values
	attributeValues, err := utils.ConvertToDynamoDBAttributeValues(item)
	if err != nil {
		return fmt.Errorf("failed to convert item to DynamoDB format: %w", err)
	}

	// Create the put item input
	input := &dynamodb.PutItemInput{
		TableName: &tableName,
		Item:      attributeValues,
	}

	// Put the item in DynamoDB
	_, err = db.PutItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put item in DynamoDB: %w", err)
	}

	return nil
}

// GetItem retrieves an item from DynamoDB
func GetItem(ctx context.Context, db *dynamodb.Client, tableName string, key map[string]interface{}) (map[string]interface{}, error) {
	// Convert the key to DynamoDB attribute values
	keyAttributes, err := utils.ConvertToDynamoDBAttributeValues(key)
	if err != nil {
		return nil, fmt.Errorf("failed to convert key to DynamoDB format: %w", err)
	}

	// Create the get item input
	input := &dynamodb.GetItemInput{
		TableName: &tableName,
		Key:       keyAttributes,
	}

	// Get the item from DynamoDB
	result, err := db.GetItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get item from DynamoDB: %w", err)
	}

	// If the item doesn't exist, return nil
	if len(result.Item) == 0 {
		return nil, nil
	}

	// Convert the DynamoDB attribute values to a map
	item, err := utils.ConvertFromDynamoDBAttributeValues(result.Item)
	if err != nil {
		return nil, fmt.Errorf("failed to convert item from DynamoDB format: %w", err)
	}

	return item, nil
}

// DeleteItem deletes an item from DynamoDB
func DeleteItem(ctx context.Context, db *dynamodb.Client, tableName string, key map[string]interface{}) error {
	// Convert the key to DynamoDB attribute values
	keyAttributes, err := utils.ConvertToDynamoDBAttributeValues(key)
	if err != nil {
		return fmt.Errorf("failed to convert key to DynamoDB format: %w", err)
	}

	// Create the delete item input
	input := &dynamodb.DeleteItemInput{
		TableName: &tableName,
		Key:       keyAttributes,
	}

	// Delete the item from DynamoDB
	_, err = db.DeleteItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete item from DynamoDB: %w", err)
	}

	return nil
}

// GetImagesByUserAndProduct retrieves all customization images for a specific user and product/variant
func (s *DynamoDBCustomizationService) GetImagesByUserAndProduct(ctx context.Context, userID, productID, variantID string) ([]*models.CustomizationImage, error) {
	utils.DebugLog("Getting customization images for user: %s, product: %s, variant: %s", userID, productID, variantID)

	if userID == "" {
		return nil, errors.New("userID is required")
	}

	var queryInput map[string]interface{}

	if productID != "" && variantID != "" {
		// Query by product and variant using GSI2
		queryInput = map[string]interface{}{
			"IndexName":              "GSI2",
			"KeyConditionExpression": "GSI2PK = :pk AND begins_with(GSI2SK, :sk)",
			"ExpressionAttributeValues": map[string]interface{}{
				":pk": fmt.Sprintf("PRODUCT#%s", productID),
				":sk": fmt.Sprintf("VARIANT#%s#CUSTOMIZATION#", variantID),
			},
		}
	} else {
		// Query by user using GSI1
		queryInput = map[string]interface{}{
			"IndexName":              "GSI1",
			"KeyConditionExpression": "GSI1PK = :pk AND begins_with(GSI1SK, :sk)",
			"ExpressionAttributeValues": map[string]interface{}{
				":pk": fmt.Sprintf("USER#%s", userID),
				":sk": "CUSTOMIZATION#",
			},
		}
	}

	// Add filter for user ID if querying by product/variant
	if productID != "" && variantID != "" {
		queryInput["FilterExpression"] = "UserID = :userId"
		queryInput["ExpressionAttributeValues"].(map[string]interface{})[":userId"] = userID
	}

	// Execute the query
	items, err := utils.QueryItems(ctx, s.db, s.tableName, queryInput)
	if err != nil {
		utils.ErrorLog("Failed to query customization images: %v", err)
		return nil, fmt.Errorf("failed to query customization images: %w", err)
	}

	// Convert the items to CustomizationImage models
	images := make([]*models.CustomizationImage, 0, len(items))
	for _, item := range items {
		imageID, ok := item["ID"].(string)
		if !ok {
			utils.ErrorLog("Item missing ID field: %v", item)
			continue
		}

		// Get the image details
		image, err := s.GetImage(ctx, imageID)
		if err != nil {
			utils.ErrorLog("Failed to get image details for ID %s: %v", imageID, err)
			continue
		}

		images = append(images, image)
	}

	utils.DebugLog("Found %d customization images for user %s", len(images), userID)
	return images, nil
}

// LinkImageToCartItem links a customization image to a specific cart item
func (s *DynamoDBCustomizationService) LinkImageToCartItem(ctx context.Context, imageID, cartID, cartItemID string) error {
	utils.DebugLog("Linking customization image %s to cart item %s in cart %s", imageID, cartItemID, cartID)

	// First, get the image to verify it exists
	_, err := s.GetImage(ctx, imageID)
	if err != nil {
		return fmt.Errorf("failed to get image: %w", err)
	}

	// Update the image record to include the cart item ID
	key := map[string]interface{}{
		"PK": fmt.Sprintf("CUSTOMIZATION#%s", imageID),
		"SK": "METADATA",
	}

	// For now, just re-store the entire item with the updates
	// Get the full item
	item, err := GetItem(ctx, s.db, s.tableName, key)
	if err != nil {
		return fmt.Errorf("failed to get image metadata: %w", err)
	}

	if item == nil {
		return ErrObjectNotFound
	}

	// Update the item with new values
	item["CartID"] = cartID
	item["CartItemID"] = cartItemID
	item["GSI3PK"] = fmt.Sprintf("CART#%s", cartID)
	item["GSI3SK"] = fmt.Sprintf("ITEM#%s#CUSTOMIZATION#%s", cartItemID, imageID)

	// Store the updated item
	err = PutItem(ctx, s.db, s.tableName, item)
	if err != nil {
		return fmt.Errorf("failed to update image metadata: %w", err)
	}

	utils.DebugLog("Successfully linked image %s to cart item %s", imageID, cartItemID)
	return nil
}
