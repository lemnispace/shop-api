package models

import (
	"time"
)

// CustomizationImage represents a user-uploaded image for product customization
type CustomizationImage struct {
	ID          string            `json:"id"`
	URL         string            `json:"url"`
	Width       int               `json:"width"`
	Height      int               `json:"height"`
	ContentType string            `json:"contentType"`
	Size        int64             `json:"size"`
	BucketName  string            `json:"-"`                // S3 bucket name (not exposed in API)
	ObjectKey   string            `json:"-"`                // S3 object key (not exposed in API)
	UserID      string            `json:"userId,omitempty"` // The ID of the user who uploaded the image
	CartID      string            `json:"cartId,omitempty"`
	ProductID   string            `json:"productId,omitempty"`
	VariantID   string            `json:"variantId,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	ExpiresAt   time.Time         `json:"expiresAt"`
}

// ImageOperation represents an operation to be performed on an image
type ImageOperation struct {
	Type                string `json:"type"` // "removeBackground", "resize", "crop"
	Width               int    `json:"width,omitempty"`
	Height              int    `json:"height,omitempty"`
	MaintainAspectRatio bool   `json:"maintainAspectRatio,omitempty"`
	X                   int    `json:"x,omitempty"`
	Y                   int    `json:"y,omitempty"`
}

// ProcessImageRequest represents a request to process an uploaded image
type ProcessImageRequest struct {
	Operations []ImageOperation `json:"operations"`
}

// ProcessImageResponse represents the response from processing an image
type ProcessImageResponse struct {
	ID              string    `json:"id"`
	OriginalImageID string    `json:"originalImageId"`
	URL             string    `json:"url"`
	Width           int       `json:"width"`
	Height          int       `json:"height"`
	ContentType     string    `json:"contentType"`
	Size            int64     `json:"size"`
	UserID          string    `json:"userId,omitempty"` // The ID of the user who created the image
	CreatedAt       time.Time `json:"createdAt"`
	ExpiresAt       time.Time `json:"expiresAt"`
}
