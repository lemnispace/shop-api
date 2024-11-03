package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/internal/utils"
)

// ProductsHandler handles requests to /v1/products
func ProductsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ListAllProducts(w, r)
	case http.MethodPost:
		CreateProduct(w, r)
	default:
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// ProductDetailHandler handles requests to /v1/products/{productId}
func ProductDetailHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	apiPrefix := "/v1/products/"
	if !strings.HasPrefix(path, apiPrefix) {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid path")
		return
	}

	// Extract productId from URL
	productId := strings.TrimPrefix(path, apiPrefix)
	productIdParts := strings.Split(productId, "/")
	if len(productIdParts) == 0 || productIdParts[0] == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Product ID is required")
		return
	}
	productId = productIdParts[0]

	switch r.Method {
	case http.MethodGet:
		GetProduct(w, r, productId)
	case http.MethodPut:
		UpdateProduct(w, r, productId)
	case http.MethodDelete:
		DeleteProduct(w, r, productId)
	default:
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func ListAllProducts(w http.ResponseWriter, r *http.Request) {
	// Placeholder for fetching products
	var products []models.Product

	// TODO: Implement logic to retrieve products from database

	utils.JSONResponse(w, http.StatusOK, products)
}

func CreateProduct(w http.ResponseWriter, r *http.Request) {
	// Decode the request body
	var input models.ProductInput
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// TODO: Implement logic to create a new product

	var product models.Product
	// Set product fields from input...

	utils.JSONResponse(w, http.StatusCreated, product)
}

func GetProduct(w http.ResponseWriter, r *http.Request, productId string) {
	// TODO: Implement logic to retrieve product by ID

	var product models.Product
	// Retrieve product...

	utils.JSONResponse(w, http.StatusOK, product)
}

func UpdateProduct(w http.ResponseWriter, r *http.Request, productId string) {
	// Decode the request body
	var input models.ProductInput
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// TODO: Implement logic to update the product

	var product models.Product
	// Update product fields...

	utils.JSONResponse(w, http.StatusOK, product)
}

func DeleteProduct(w http.ResponseWriter, r *http.Request, productId string) {
	// TODO: Implement logic to delete the product

	w.WriteHeader(http.StatusNoContent)
}
