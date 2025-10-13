package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lemnispace/shop-api/internal/middleware"
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

// customizationErrorResponse provides a consistent error response structure for customization operations
func customizationErrorResponse(c *gin.Context, err error, defaultStatusCode int, defaultMessage string) {
	statusCode := defaultStatusCode
	message := defaultMessage

	switch err {
	case services.ErrImageNotFound:
		statusCode = http.StatusNotFound
		message = "Image not found"
	case services.ErrInvalidOperation:
		statusCode = http.StatusBadRequest
		message = "Invalid image processing operation requested"
	default:
		if err != nil {
			message = err.Error()
		}
	}

	utils.ErrorResponseWithDetails(c, statusCode, message, nil)
}

// UploadCustomizationImage handles POST /v1/customizations/images
func UploadCustomizationImage(c *gin.Context) {
	if customizationService == nil {
		utils.ErrorResponse(c, http.StatusServiceUnavailable, "Customization service not initialized")
		return
	}

	// Get authenticated customer ID from JWT
	customerID, exists := middleware.GetCustomerID(c)
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Authentication required")
		return
	}

	c.Request.ParseMultipartForm(maxUploadSize) // Handled by Gin binding

	fileHeader, err := c.FormFile("image")
	if err != nil {
		customizationErrorResponse(c, err, http.StatusBadRequest, "Missing or invalid image file")
		return
	}

	// Check file size (optional, can rely on server limits)
	if fileHeader.Size > maxUploadSize {
		customizationErrorResponse(c, nil, http.StatusRequestEntityTooLarge, "File too large")
		return
	}

	// Get optional parameters from form data
	cartID := c.PostForm("cartId")
	productID := c.PostForm("productId")
	variantID := c.PostForm("variantId")

	// Open the file
	file, err := fileHeader.Open()
	if err != nil {
		customizationErrorResponse(c, err, http.StatusInternalServerError, "Failed to open uploaded file")
		return
	}
	defer file.Close()

	// Process the upload via service (using authenticated customerID)
	image, err := customizationService.UploadImage(c.Request.Context(), file, fileHeader, customerID, cartID, productID, variantID)
	if err != nil {
		customizationErrorResponse(c, err, http.StatusInternalServerError, "Failed to upload image")
		return
	}

	// Return the result (ensure model matches API spec)
	utils.JSONResponse(c, http.StatusCreated, image)
}

// GetCustomizationImage handles GET /v1/customizations/images/:imageId
// Note: This endpoint is not in API_DESIGN.md
func GetCustomizationImage(c *gin.Context) {
	if customizationService == nil {
		utils.ErrorResponse(c, http.StatusServiceUnavailable, "Customization service not initialized")
		return
	}

	// Get authenticated customer ID from JWT
	customerID, exists := middleware.GetCustomerID(c)
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Authentication required")
		return
	}

	imageID := c.Param("imageId")
	if imageID == "" {
		customizationErrorResponse(c, nil, http.StatusBadRequest, "Image ID required")
		return
	}

	image, err := customizationService.GetImage(c.Request.Context(), imageID)
	if err != nil {
		customizationErrorResponse(c, err, http.StatusInternalServerError, fmt.Sprintf("Failed to get image with ID %s", imageID))
		return
	}

	// Verify image belongs to authenticated customer
	if image.UserID != customerID {
		customizationErrorResponse(c, nil, http.StatusForbidden, "Unauthorized - image does not belong to authenticated user")
		return
	}

	utils.JSONResponse(c, http.StatusOK, image)
}

// ListCustomizationImages handles GET /v1/customizations/images
// Note: This endpoint is not in API_DESIGN.md
func ListCustomizationImages(c *gin.Context) {
	if customizationService == nil {
		utils.ErrorResponse(c, http.StatusServiceUnavailable, "Customization service not initialized")
		return
	}

	// Get authenticated customer ID from JWT
	customerID, exists := middleware.GetCustomerID(c)
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Get query parameters for filtering
	productID := c.Query("productId")
	variantID := c.Query("variantId")
	// Note: Pagination is not currently supported by the service layer.
	// For future enhancement, add limit/cursor parameters to GetImagesByUserAndProduct.

	images, err := customizationService.GetImagesByUserAndProduct(c.Request.Context(), customerID, productID, variantID)
	if err != nil {
		customizationErrorResponse(c, err, http.StatusInternalServerError, "Failed to list images")
		return
	}

	// Return only images, no pagination as it's not supported by the service layer currently
	utils.JSONResponse(c, http.StatusOK, gin.H{
		"images": images,
	})
}

// ProcessCustomizationImage handles POST /v1/customizations/images/:imageId/process
func ProcessCustomizationImage(c *gin.Context) {
	if customizationService == nil {
		utils.ErrorResponse(c, http.StatusServiceUnavailable, "Customization service not initialized")
		return
	}

	// Get authenticated customer ID from JWT
	customerID, exists := middleware.GetCustomerID(c)
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Authentication required")
		return
	}

	imageID := c.Param("imageId")
	if imageID == "" {
		customizationErrorResponse(c, nil, http.StatusBadRequest, "Image ID required")
		return
	}

	// Verify image exists and belongs to authenticated customer
	image, err := customizationService.GetImage(c.Request.Context(), imageID)
	if err != nil {
		customizationErrorResponse(c, err, http.StatusNotFound, fmt.Sprintf("Image %s not found", imageID))
		return
	}
	if image.UserID != customerID {
		customizationErrorResponse(c, nil, http.StatusForbidden, "Unauthorized - image does not belong to authenticated user")
		return
	}

	var request models.ProcessImageRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		customizationErrorResponse(c, err, http.StatusBadRequest, "Invalid request format")
		return
	}

	if len(request.Operations) == 0 {
		customizationErrorResponse(c, nil, http.StatusBadRequest, "No operations specified")
		return
	}

	response, err := customizationService.ProcessImage(c.Request.Context(), imageID, request)
	if err != nil {
		customizationErrorResponse(c, err, http.StatusInternalServerError, fmt.Sprintf("Failed to process image %s: %v", imageID, err))
		return
	}

	utils.JSONResponse(c, http.StatusOK, response)
}

// DeleteCustomizationImage handles DELETE /v1/customizations/images/:imageId
// Note: This endpoint is not in API_DESIGN.md
func DeleteCustomizationImage(c *gin.Context) {
	if customizationService == nil {
		utils.ErrorResponse(c, http.StatusServiceUnavailable, "Customization service not initialized")
		return
	}

	// Get authenticated customer ID from JWT
	customerID, exists := middleware.GetCustomerID(c)
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Authentication required")
		return
	}

	imageID := c.Param("imageId")
	if imageID == "" {
		customizationErrorResponse(c, nil, http.StatusBadRequest, "Image ID required")
		return
	}

	// Verify ownership before deleting
	image, err := customizationService.GetImage(c.Request.Context(), imageID)
	if err != nil {
		if err == services.ErrImageNotFound {
			utils.NoContent(c) // Idempotent: Already deleted or never existed
			return
		}
		customizationErrorResponse(c, err, http.StatusInternalServerError, "Failed to verify image before delete")
		return
	}
	if image.UserID != customerID {
		customizationErrorResponse(c, nil, http.StatusForbidden, "Unauthorized - image does not belong to authenticated user")
		return
	}

	err = customizationService.DeleteImage(c.Request.Context(), imageID)
	if err != nil {
		// ErrImageNotFound already handled above
		customizationErrorResponse(c, err, http.StatusInternalServerError, "Failed to delete image")
		return
	}

	utils.NoContent(c)
}

// LinkImageToCartItem handles POST /v1/customizations/images/:imageId/link
// Note: This endpoint is not in API_DESIGN.md
func LinkImageToCartItem(c *gin.Context) {
	if customizationService == nil {
		utils.ErrorResponse(c, http.StatusServiceUnavailable, "Customization service not initialized")
		return
	}

	// Get authenticated customer ID from JWT
	customerID, exists := middleware.GetCustomerID(c)
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Authentication required")
		return
	}

	imageID := c.Param("imageId")
	if imageID == "" {
		customizationErrorResponse(c, nil, http.StatusBadRequest, "Image ID required")
		return
	}

	var input struct {
		CartID string `json:"cartId" binding:"required"`
		ItemID string `json:"itemId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		customizationErrorResponse(c, err, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Verify ownership
	image, err := customizationService.GetImage(c.Request.Context(), imageID)
	if err != nil {
		customizationErrorResponse(c, err, http.StatusNotFound, "Image not found")
		return
	}
	if image.UserID != customerID {
		customizationErrorResponse(c, nil, http.StatusForbidden, "Unauthorized - image does not belong to authenticated user")
		return
	}

	err = customizationService.LinkImageToCartItem(c.Request.Context(), imageID, input.CartID, input.ItemID)
	if err != nil {
		customizationErrorResponse(c, err, http.StatusInternalServerError, "Failed to link image to cart item")
		return
	}

	utils.JSONResponse(c, http.StatusOK, gin.H{"success": true})
}
