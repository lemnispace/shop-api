package models

import "time"

// Dimensions represents the physical dimensions of a product.
type Dimensions struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Depth  float64 `json:"depth"`
	Length float64 `json:"length"`
	Weight float64 `json:"weight"`
}

// FulfillmentData represents fulfillment-related data for a product.
type FulfillmentData struct {
	PartnerID        string                 `json:"partnerId"`
	PartnerProductID string                 `json:"partnerProductId"`
	PartnerVariantID string                 `json:"partnerVariantId"`
	AdditionalData   map[string]interface{} `json:"additionalData"`
	HSCode           string                 `json:"hsCode"`
	CountryOfOrigin  string                 `json:"countryOfOrigin"`
	Harmonized       bool                   `json:"harmonized"`
	RequiresShipping bool                   `json:"requiresShipping"`
}

// VariantOption represents an option for a product variant.
type VariantOption struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Image represents an image for a product.
type Image struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	AltText   string    `json:"altText"`
	IsDefault bool      `json:"isDefault"`
	Variants  []string  `json:"variants"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ImageInput represents the data required to create or update an image
type ImageInput struct {
	URL       string   `json:"url"`
	AltText   string   `json:"altText"`
	IsDefault bool     `json:"isDefault"`
	Variants  []string `json:"variants"` // IDs of associated variants
	Position  int      `json:"position"`
}

// Address represents a physical address.
type Address struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Company   string `json:"company"`
	Address1  string `json:"address1"`
	Address2  string `json:"address2"`
	City      string `json:"city"`
	Province  string `json:"province"`
	Country   string `json:"country"`
	Zip       string `json:"zip"`
	Phone     string `json:"phone"`
}

// PaginationLinks represents the next/prev/self links for paginated responses
type PaginationLinks struct {
	Next string `json:"next,omitempty"`
	Prev string `json:"prev,omitempty"`
	Self string `json:"self"`
}

// PaginatedResponse is a generic paginated response structure
type PaginatedResponse struct {
	Items []interface{}   `json:"items"`
	Links PaginationLinks `json:"links"`
}
