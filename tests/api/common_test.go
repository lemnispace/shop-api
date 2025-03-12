package api

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/internal/routers"
	"github.com/lemnispace/shop-api/internal/services"
	"github.com/lemnispace/shop-api/internal/utils"
	"github.com/stretchr/testify/mock"
)

// MockCustomizationService is a mock implementation of the CustomizationService interface for testing
type MockCustomizationService struct {
	mock.Mock
}

func (m *MockCustomizationService) UploadImage(ctx context.Context, file multipart.File, fileHeader *multipart.FileHeader, userID, cartID, productID, variantID string) (*models.CustomizationImage, error) {
	args := m.Called(ctx, file, fileHeader, userID, cartID, productID, variantID)
	return args.Get(0).(*models.CustomizationImage), args.Error(1)
}

func (m *MockCustomizationService) GetImage(ctx context.Context, imageID string) (*models.CustomizationImage, error) {
	args := m.Called(ctx, imageID)
	return args.Get(0).(*models.CustomizationImage), args.Error(1)
}

func (m *MockCustomizationService) ProcessImage(ctx context.Context, imageID string, request models.ProcessImageRequest) (*models.ProcessImageResponse, error) {
	args := m.Called(ctx, imageID, request)
	return args.Get(0).(*models.ProcessImageResponse), args.Error(1)
}

func (m *MockCustomizationService) DeleteImage(ctx context.Context, imageID string) error {
	args := m.Called(ctx, imageID)
	return args.Error(0)
}

func (m *MockCustomizationService) GetImagesByUserAndProduct(ctx context.Context, userID, productID, variantID string) ([]*models.CustomizationImage, error) {
	args := m.Called(ctx, userID, productID, variantID)
	return args.Get(0).([]*models.CustomizationImage), args.Error(1)
}

func (m *MockCustomizationService) LinkImageToCartItem(ctx context.Context, imageID, cartID, cartItemID string) error {
	args := m.Called(ctx, imageID, cartID, cartItemID)
	return args.Error(0)
}

// setupTestRouter creates a router with test configurations
func setupTestRouter() http.Handler {
	// Use local DynamoDB for tests
	endpoint := os.Getenv("DYNAMODB_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8000"
	}

	// Configure AWS SDK
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithEndpointResolver(aws.EndpointResolverFunc(
			func(service, region string) (aws.Endpoint, error) {
				if service == "dynamodb" {
					return aws.Endpoint{
						URL:         endpoint,
						SigningName: "dynamodb",
					}, nil
				}
				return aws.Endpoint{}, nil
			})),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	if err != nil {
		panic(err)
	}

	// Create DynamoDB client
	db := dynamodb.NewFromConfig(cfg)

	// Create mock S3 service
	mockS3 := new(MockS3Service)

	// Create mock customization service
	mockCustomizationService := new(MockCustomizationService)

	// Set up default mock behaviors for customization service
	mockCustomizationService.On("UploadImage", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&models.CustomizationImage{
			ID:          GenerateID(),
			URL:         "https://test-url.com/test-image.jpg",
			Width:       100,
			Height:      100,
			ContentType: "image/jpeg",
			Size:        1024,
			BucketName:  "test-bucket",
			ObjectKey:   "test-key.jpg",
			UserID:      "user123",
			ProductID:   "test-product-123",
			VariantID:   "test-variant-456",
			CartID:      "test-cart-789",
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(7 * 24 * time.Hour),
		}, nil)

	mockCustomizationService.On("GetImage", mock.Anything, mock.Anything).
		Return(&models.CustomizationImage{
			ID:          "test-image-id",
			URL:         "https://test-url.com/test-image.jpg",
			Width:       100,
			Height:      100,
			ContentType: "image/jpeg",
			Size:        1024,
			BucketName:  "test-bucket",
			ObjectKey:   "test-key.jpg",
			UserID:      "user123", // This will be checked for authorization
			ProductID:   "test-product-123",
			VariantID:   "test-variant-456",
			CartID:      "test-cart-789",
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(7 * 24 * time.Hour),
		}, nil)

	mockCustomizationService.On("ProcessImage", mock.Anything, mock.Anything, mock.Anything).
		Return(&models.ProcessImageResponse{
			ID:              GenerateID(),
			OriginalImageID: "test-image-id",
			URL:             "https://test-url.com/processed-image.jpg",
			Width:           50,
			Height:          50,
			ContentType:     "image/jpeg",
			Size:            512,
			UserID:          "user123",
			CreatedAt:       time.Now(),
			ExpiresAt:       time.Now().Add(7 * 24 * time.Hour),
		}, nil)

	mockCustomizationService.On("DeleteImage", mock.Anything, mock.Anything).Return(nil)

	mockCustomizationService.On("GetImagesByUserAndProduct", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]*models.CustomizationImage{
			{
				ID:          GenerateID(),
				URL:         "https://test-url.com/test-image.jpg",
				Width:       100,
				Height:      100,
				ContentType: "image/jpeg",
				Size:        1024,
				UserID:      "user123",
				ProductID:   "test-product-123",
				VariantID:   "test-variant-456",
				CreatedAt:   time.Now(),
				ExpiresAt:   time.Now().Add(7 * 24 * time.Hour),
			},
		}, nil)

	mockCustomizationService.On("LinkImageToCartItem", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Create service factory
	serviceFactory := func() (services.ProductService, services.CollectionService, *services.CartService, services.S3Service, services.CustomizationService) {
		// Use test table name
		tableName := os.Getenv("DYNAMODB_TABLE")
		if tableName == "" {
			tableName = "test-table"
		}

		productService := services.NewProductService(db, tableName)
		collectionService := services.NewCollectionService(db, tableName, productService)
		cartService := services.NewCartService(db, productService, tableName)

		return productService, collectionService, cartService, mockS3, mockCustomizationService
	}

	// Set up router with our service factory
	routers.SetServiceFactory(serviceFactory)

	// Create a custom router for testing
	router := http.NewServeMux()

	// Register handlers
	apiPrefix := "/v1"
	router.HandleFunc(apiPrefix+"/customizations/images", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			// Handle image upload
			image, _ := mockCustomizationService.UploadImage(r.Context(), nil, &multipart.FileHeader{}, r.FormValue("userId"), r.FormValue("cartId"), r.FormValue("productId"), r.FormValue("variantId"))
			utils.SendJSONResponse(w, http.StatusCreated, image)
		case http.MethodGet:
			// Handle list images
			images, _ := mockCustomizationService.GetImagesByUserAndProduct(r.Context(), r.URL.Query().Get("userId"), r.URL.Query().Get("productId"), r.URL.Query().Get("variantId"))
			utils.SendJSONResponse(w, http.StatusOK, images)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	router.HandleFunc(apiPrefix+"/customizations/images/", func(w http.ResponseWriter, r *http.Request) {
		// Extract the image ID and operation from the URL
		path := strings.TrimPrefix(r.URL.Path, apiPrefix+"/customizations/images/")
		parts := strings.Split(path, "/")

		if len(parts) == 0 || parts[0] == "" {
			http.Error(w, "Invalid URL", http.StatusBadRequest)
			return
		}

		imageID := parts[0]

		// Check if this is a process request
		if len(parts) > 1 && parts[1] == "process" {
			// Handle process image
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			// Check user authorization
			userID := r.URL.Query().Get("userId")
			image, _ := mockCustomizationService.GetImage(r.Context(), imageID)
			if image.UserID != userID {
				http.Error(w, "Unauthorized", http.StatusForbidden)
				return
			}

			// Process the image
			var request models.ProcessImageRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, "Invalid request format", http.StatusBadRequest)
				return
			}

			response, _ := mockCustomizationService.ProcessImage(r.Context(), imageID, request)
			utils.SendJSONResponse(w, http.StatusOK, response)
			return
		}

		// Handle regular image operations
		switch r.Method {
		case http.MethodGet:
			// Check user authorization
			userID := r.URL.Query().Get("userId")
			image, _ := mockCustomizationService.GetImage(r.Context(), imageID)
			if image.UserID != userID {
				http.Error(w, "Unauthorized", http.StatusForbidden)
				return
			}

			utils.SendJSONResponse(w, http.StatusOK, image)
		case http.MethodDelete:
			// Check user authorization
			userID := r.URL.Query().Get("userId")
			image, _ := mockCustomizationService.GetImage(r.Context(), imageID)
			if image.UserID != userID {
				http.Error(w, "Unauthorized", http.StatusForbidden)
				return
			}

			mockCustomizationService.DeleteImage(r.Context(), imageID)
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return router
}

// MockS3Service is a mock implementation of the S3Service interface for testing
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

func (m *MockS3Service) GeneratePresignedUploadURL(ctx context.Context, bucketName, objectKey, contentType string, expiration time.Duration) (string, error) {
	args := m.Called(ctx, bucketName, objectKey, contentType, expiration)
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

// GenerateID generates a unique ID for testing
func GenerateID() string {
	return uuid.New().String()
}
