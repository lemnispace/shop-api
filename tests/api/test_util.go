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
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/lemnispace/shop-api/internal/handlers"
	"github.com/lemnispace/shop-api/internal/services"
	"github.com/lemnispace/shop-api/tests"
)

// apiPrefix for all API requests
const apiPrefix = "/v1"

var (
	// TestServer is the httptest server used for testing
	TestServer     *httptest.Server
	testRouter     *http.ServeMux
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
	testRouter = http.NewServeMux()

	// Initialize services with DynamoDB
	var err error
	productService, err = tests.CreateTestProductService()
	if err != nil {
		log.Fatalf("Failed to setup product service: %v", err)
	}
	handlers.SetProductService(productService)

	collectionService, err := tests.CreateTestCollectionService(productService)
	if err != nil {
		log.Fatalf("Failed to setup collection service: %v", err)
	}
	handlers.SetCollectionService(collectionService)

	cartService := services.NewCartService(dynamoClient, productService, tests.TestTableName())
	handlers.SetCartService(cartService)

	// Set up routes
	testRouter.HandleFunc(apiPrefix+"/products", handlers.ProductsHandler)
	testRouter.HandleFunc(apiPrefix+"/products/", handlers.ProductDetailHandler)
	testRouter.HandleFunc(apiPrefix+"/products/count", handlers.ProductCountHandler)
	testRouter.HandleFunc(apiPrefix+"/products/variants", handlers.ProductVariantsHandler)
	testRouter.HandleFunc(apiPrefix+"/collections", handlers.CollectionsHandler)
	testRouter.HandleFunc(apiPrefix+"/collections/", handlers.CollectionDetailHandler)
	testRouter.HandleFunc(apiPrefix+"/collections/count", handlers.CollectionCountHandler)
	testRouter.HandleFunc(apiPrefix+"/cart", handlers.CartHandler)
	testRouter.HandleFunc(apiPrefix+"/cart/", handlers.CartDetailHandler)

	// Create a test server with the router
	TestServer = httptest.NewServer(testRouter)
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

// MakeRequest sends an HTTP request to the test server and returns the response and body
func MakeRequest(t *testing.T, method, path string, body interface{}) (*http.Response, string) {
	t.Helper()

	var req *http.Request
	var err error

	// Make test server if it doesn't exist
	if TestServer == nil {
		SetupTestServer()
	}

	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}

		req = httptest.NewRequest(method, path, bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	t.Logf("Making request: %s %s", method, path)

	// Record response
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	resp := w.Result()

	// Read body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	defer resp.Body.Close()

	bodyString := string(bodyBytes)
	t.Logf("Response status: %d", resp.StatusCode)

	// Allow a small delay for DynamoDB operations to propagate
	// This helps with eventual consistency issues in tests
	if method != "GET" && (resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK) {
		time.Sleep(100 * time.Millisecond)
	}

	return resp, bodyString
}

// ParseJSONResponse parses a JSON response body into a result object
func ParseJSONResponse(t *testing.T, responseBody string, result interface{}) {
	t.Helper()
	err := json.Unmarshal([]byte(responseBody), result)
	if err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
}

// TestMain runs before and after all tests in the package
func TestMain(m *testing.M) {
	// Run all the tests
	exitCode := m.Run()

	// Always run cleanup after all tests complete
	tests.FinalCleanup()

	// Exit with the same code from the tests
	os.Exit(exitCode)
}
