package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/lemnispace/shop-api/internal/handlers"
	"github.com/lemnispace/shop-api/internal/services"
)

const (
	apiPrefix = "/v1"
)

var (
	// TestServer is the httptest server used for testing
	TestServer     *httptest.Server
	productService services.ProductService
)

// init sets up the test environment
func init() {
	SetupTestServer()
}

// SetupTestServer initializes the test server with routes
func SetupTestServer() {
	// Initialize the router for testing
	router := http.NewServeMux()

	// Initialize in-memory product service
	productService = services.NewInMemoryProductService()
	handlers.SetProductService(productService)

	// Initialize in-memory collection service
	collectionService := services.NewInMemoryCollectionService(productService)
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

	// Create test server
	TestServer = httptest.NewServer(router)
}

// TeardownTestServer closes the test server
func TeardownTestServer() {
	if TestServer != nil {
		TestServer.Close()
	}
}

// MakeRequest is a helper function to make HTTP requests to the test server
func MakeRequest(t *testing.T, method, path string, body interface{}) (*http.Response, []byte) {
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

	// Make request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

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
