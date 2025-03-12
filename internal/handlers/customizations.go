package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/internal/services"
	"github.com/lemnispace/shop-api/internal/utils"
)

// Maximum file size for image upload (10MB)
const maxUploadSize = 10 * 1024 * 1024

var customizationService services.CustomizationService

// SetCustomizationService sets the customization service to be used by the handlers
func SetCustomizationService(service services.CustomizationService) {
	customizationService = service
}

// CustomizationsHandler handles requests to /customizations/images
func CustomizationsHandler(w http.ResponseWriter, r *http.Request) {
	if customizationService == nil {
		utils.ErrorLog("Customization service not initialized")
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodPost:
		uploadCustomizationImage(w, r)
	case http.MethodGet:
		listCustomizationImages(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// CustomizationDetailHandler handles requests to /customizations/images/{imageId}
func CustomizationDetailHandler(w http.ResponseWriter, r *http.Request) {
	if customizationService == nil {
		utils.ErrorLog("Customization service not initialized")
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Extract the image ID from the URL
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	imageID := parts[3]
	if imageID == "" {
		http.Error(w, "Image ID required", http.StatusBadRequest)
		return
	}

	// Check if this is a request to process an image
	if len(parts) >= 5 && parts[4] == "process" {
		processCustomizationImage(w, r, imageID)
		return
	}

	// Check if this is a request to link image to cart item
	if len(parts) >= 5 && parts[4] == "link" {
		linkImageToCartItem(w, r, imageID)
		return
	}

	// Otherwise, handle regular image operations
	switch r.Method {
	case http.MethodGet:
		getCustomizationImage(w, r, imageID)
	case http.MethodDelete:
		deleteCustomizationImage(w, r, imageID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// uploadCustomizationImage handles POST /customizations/images
func uploadCustomizationImage(w http.ResponseWriter, r *http.Request) {
	utils.DebugLog("Handling image upload request")

	// Limit the request size
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	// Parse the multipart form
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		utils.ErrorLog("Failed to parse multipart form: %v", err)
		http.Error(w, "File too large or invalid form", http.StatusBadRequest)
		return
	}

	// Get the file from the form
	file, fileHeader, err := r.FormFile("image")
	if err != nil {
		utils.ErrorLog("Failed to get file from form: %v", err)
		http.Error(w, "Missing or invalid image file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get parameters from the form
	userID := r.FormValue("userId")
	cartID := r.FormValue("cartId")
	productID := r.FormValue("productId")
	variantID := r.FormValue("variantId")

	// Require userID for user-specific customizations
	if userID == "" {
		utils.ErrorLog("Missing userID parameter")
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Validate that productID and variantID are both provided or both empty
	if (productID == "" && variantID != "") || (productID != "" && variantID == "") {
		utils.ErrorLog("When associating with a product, both productID and variantID must be provided")
		http.Error(w, "Both product ID and variant ID must be provided when associating with a product", http.StatusBadRequest)
		return
	}

	// Process the upload
	image, err := customizationService.UploadImage(r.Context(), file, fileHeader, userID, cartID, productID, variantID)
	if err != nil {
		handleCustomizationError(w, err, "Failed to upload image")
		return
	}

	// Return the result
	utils.SendJSONResponse(w, http.StatusCreated, image)
}

// getCustomizationImage handles GET /customizations/images/{imageId}
func getCustomizationImage(w http.ResponseWriter, r *http.Request, imageID string) {
	utils.DebugLog("Getting customization image - ID: %s", imageID)

	// Extract userID from query parameters for user validation
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		utils.ErrorLog("User ID not provided for image access")
		return
	}

	image, err := customizationService.GetImage(r.Context(), imageID)
	if err != nil {
		handleCustomizationError(w, err, fmt.Sprintf("Failed to get image with ID %s", imageID))
		return
	}

	// Ensure users can only access their own customizations
	if image.UserID != userID {
		utils.ErrorLog("Unauthorized access attempt to image %s by user %s (owner: %s)", imageID, userID, image.UserID)
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, image)
}

// listCustomizationImages handles GET /customizations/images
func listCustomizationImages(w http.ResponseWriter, r *http.Request) {
	utils.DebugLog("Listing customization images")

	// Get query parameters
	userID := r.URL.Query().Get("userId")
	productID := r.URL.Query().Get("productId")
	variantID := r.URL.Query().Get("variantId")

	// Require userID for user-specific customizations
	if userID == "" {
		utils.ErrorLog("Missing userID parameter")
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Get the images
	images, err := customizationService.GetImagesByUserAndProduct(r.Context(), userID, productID, variantID)
	if err != nil {
		handleCustomizationError(w, err, "Failed to list images")
		return
	}

	// Return the result
	utils.SendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"images": images,
		"count":  len(images),
	})
}

// processCustomizationImage handles POST /customizations/images/{imageId}/process
func processCustomizationImage(w http.ResponseWriter, r *http.Request, imageID string) {
	utils.DebugLog("Processing customization image - ID: %s", imageID)

	// Parse the request body
	var request models.ProcessImageRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Extract userID from query parameters for user validation
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		utils.ErrorLog("User ID not provided for image processing")
		return
	}

	// Verify the image exists
	image, err := customizationService.GetImage(r.Context(), imageID)
	if err != nil {
		handleCustomizationError(w, err, fmt.Sprintf("Failed to get image with ID %s", imageID))
		return
	}

	// Ensure users can only process their own customizations
	if image.UserID != userID {
		utils.ErrorLog("Unauthorized processing attempt for image %s by user %s (owner: %s)", imageID, userID, image.UserID)
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Validate the operations
	if len(request.Operations) == 0 {
		http.Error(w, "No operations specified", http.StatusBadRequest)
		return
	}

	response, err := customizationService.ProcessImage(r.Context(), imageID, request)
	if err != nil {
		handleCustomizationError(w, err, fmt.Sprintf("Failed to process image with ID %s", imageID))
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, response)
}

// deleteCustomizationImage handles DELETE /customizations/images/{imageId}
func deleteCustomizationImage(w http.ResponseWriter, r *http.Request, imageID string) {
	utils.DebugLog("Deleting customization image - ID: %s", imageID)

	// Extract userID from query parameters for user validation
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		utils.ErrorLog("User ID not provided for image deletion")
		return
	}

	// Verify the image exists
	image, err := customizationService.GetImage(r.Context(), imageID)
	if err != nil {
		handleCustomizationError(w, err, fmt.Sprintf("Failed to get image with ID %s", imageID))
		return
	}

	// Ensure users can only delete their own customizations
	if image.UserID != userID {
		utils.ErrorLog("Unauthorized deletion attempt for image %s by user %s (owner: %s)", imageID, userID, image.UserID)
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	err = customizationService.DeleteImage(r.Context(), imageID)
	if err != nil {
		handleCustomizationError(w, err, fmt.Sprintf("Failed to delete image with ID %s", imageID))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// linkImageToCartItem handles POST /customizations/images/{imageId}/link
func linkImageToCartItem(w http.ResponseWriter, r *http.Request, imageID string) {
	utils.DebugLog("Linking customization image to cart item - ID: %s", imageID)

	// Only allow POST method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// First, get the image to check access rights
	image, err := customizationService.GetImage(r.Context(), imageID)
	if err != nil {
		handleCustomizationError(w, err, fmt.Sprintf("Failed to get image with ID %s", imageID))
		return
	}

	// Check user access rights
	userID := r.URL.Query().Get("userId")
	if image.UserID != "" && userID != "" && image.UserID != userID {
		utils.ErrorLog("User %s attempted to link image %s belonging to user %s", userID, imageID, image.UserID)
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Parse the request body
	var request struct {
		CartID     string `json:"cartId"`
		CartItemID string `json:"cartItemId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.ErrorLog("Failed to parse link request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate the request
	if request.CartID == "" || request.CartItemID == "" {
		utils.ErrorLog("Missing required fields in link request")
		http.Error(w, "Cart ID and Cart Item ID are required", http.StatusBadRequest)
		return
	}

	// Link the image to the cart item
	err = customizationService.LinkImageToCartItem(r.Context(), imageID, request.CartID, request.CartItemID)
	if err != nil {
		handleCustomizationError(w, err, fmt.Sprintf("Failed to link image %s to cart item %s", imageID, request.CartItemID))
		return
	}

	// Return success
	utils.SendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message":    "Image linked to cart item successfully",
		"imageId":    imageID,
		"cartId":     request.CartID,
		"cartItemId": request.CartItemID,
	})
}

// handleCustomizationError handles errors from the customization service
func handleCustomizationError(w http.ResponseWriter, err error, message string) {
	utils.ErrorLog("%s: %v", message, err)

	// Map specific errors to appropriate HTTP status codes
	switch err {
	case services.ErrInvalidImage:
		http.Error(w, "Invalid image format", http.StatusBadRequest)
	case services.ErrImageTooLarge:
		http.Error(w, "Image too large", http.StatusBadRequest)
	case services.ErrOperationNotFound:
		http.Error(w, "Operation not found", http.StatusBadRequest)
	case services.ErrInvalidOperation:
		http.Error(w, "Invalid operation", http.StatusBadRequest)
	case services.ErrObjectNotFound:
		http.Error(w, "Image not found", http.StatusNotFound)
	default:
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
