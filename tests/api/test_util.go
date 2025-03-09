package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/lemnispace/shop-api/internal/handlers"
	"github.com/lemnispace/shop-api/internal/services"
	"github.com/lemnispace/shop-api/tests"
)

const (
	apiPrefix = "/v1"
)

var (
	// TestServer is the httptest server used for testing
	TestServer     *httptest.Server
	productService services.ProductService
	dynamoClient   *dynamodb.Client
)

// init sets up the test environment
func init() {
	// Set up DynamoDB
	var err error
	dynamoClient, err = tests.SetupDynamoDBForTesting()
	if err != nil {
		log.Fatalf("Failed to set up DynamoDB for testing: %v", err)
	}

	SetupTestServer()
}

// SetupTestServer initializes the test server with routes
func SetupTestServer() {
	// Initialize the router for testing
	router := http.NewServeMux()

	// Initialize services with DynamoDB
	var err error
	productService, err = tests.CreateTestProductService()
	if err != nil {
		log.Fatalf("Failed to create test product service: %v", err)
	}
	handlers.SetProductService(productService)

	collectionService, err := tests.CreateTestCollectionService(productService)
	if err != nil {
		log.Fatalf("Failed to create test collection service: %v", err)
	}
	handlers.SetCollectionService(collectionService)

	// Register product routes
	router.HandleFunc(apiPrefix+"/products", handlers.ProductsHandler)
	router.HandleFunc(apiPrefix+"/products/", handlers.ProductDetailHandler)
	router.HandleFunc(apiPrefix+"/products/count", handlers.ProductCountHandler)
	router.HandleFunc(apiPrefix+"/products/variants", handlers.ProductVariantsHandler)

	// Register collection routes
	router.HandleFunc(apiPrefix+"/collections", handlers.CollectionsHandler)
	router.HandleFunc(apiPrefix+"/collections/", handlers.CollectionDetailHandler)
	router.HandleFunc(apiPrefix+"/collections/count", handlers.CollectionCountHandler)

	// Create standard test server - simpler and less prone to errors
	TestServer = httptest.NewServer(router)
	log.Printf("Test server started at %s", TestServer.URL)
}

// TeardownTestServer closes the test server and cleans up resources
func TeardownTestServer() {
	if TestServer != nil {
		TestServer.Close()
	}

	// Clean up DynamoDB test table
	if dynamoClient != nil {
		if err := tests.TeardownDynamoDBForTesting(dynamoClient); err != nil {
			log.Printf("Warning: Failed to tear down DynamoDB: %v", err)
		}
	}
}

// MakeRequest is a helper function to make HTTP requests to the test server
func MakeRequest(t *testing.T, method, path string, body interface{}) (*http.Response, []byte) {
	// Create a client with default settings
	client := &http.Client{}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	// Create request
	req, err := http.NewRequest(method, TestServer.URL+path, reqBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Set content type for requests with body
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Log request for debugging
	t.Logf("Making request: %s %s", method, path)

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	// Log response status
	t.Logf("Response status: %d", resp.StatusCode)

	return resp, respBody
}

// ParseJSONResponse parses a JSON response body into the provided result
func ParseJSONResponse(t *testing.T, responseBody []byte, result interface{}) {
	err := json.Unmarshal(responseBody, result)
	if err != nil {
		t.Fatalf("Failed to parse response body: %v\nBody: %s", err, string(responseBody))
	}
}

// TestMain is the entry point for tests
func TestMain(m *testing.M) {
	// Setup is done in init()

	// Run tests
	code := m.Run()

	// Teardown
	TeardownTestServer()

	os.Exit(code)
}
