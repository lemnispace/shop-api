package models

import "time"

// Product represents a product in the e-commerce platform.
type Product struct {
    ID              string                 `json:"id"`
    Title           string                 `json:"title"`
    Description     string                 `json:"description"`
    Price           float64                `json:"price"`
    SKU             string                 `json:"sku"`
    Status          string                 `json:"status"` // "draft", "active", "archived"
    Inventory       int                    `json:"inventory"`
    Tags            []string               `json:"tags"`
    CustomFields    map[string]interface{} `json:"customFields"`
    Images          []Image                `json:"images"`
    Variants        []ProductVariant       `json:"variants"`
    Dimensions      Dimensions             `json:"dimensions"`
    FulfillmentData FulfillmentData        `json:"fulfillmentData"`
    CreatedAt       time.Time              `json:"createdAt"`
    UpdatedAt       time.Time              `json:"updatedAt"`
}

// ProductInput represents the data required to create or update a product.
type ProductInput struct {
    Title           string                 `json:"title"`
    Description     string                 `json:"description"`
    Price           float64                `json:"price"`
    SKU             string                 `json:"sku"`
    Status          string                 `json:"status"` // "draft", "active", "archived"
    Inventory       int                    `json:"inventory"`
    Tags            []string               `json:"tags"`
    CustomFields    map[string]interface{} `json:"customFields"`
    Variants        []ProductVariantInput  `json:"variants"`
    Dimensions      Dimensions             `json:"dimensions"`
    FulfillmentData FulfillmentData        `json:"fulfillmentData"`
}

// ProductVariant represents a variant of a product.
type ProductVariant struct {
    ID              string          `json:"id"`
    SKU             string          `json:"sku"`
    Title           string          `json:"title"`
    Price           float64         `json:"price"`
    Inventory       int             `json:"inventory"`
    Options         []VariantOption `json:"options"`
    Dimensions      Dimensions      `json:"dimensions"`
    FulfillmentData FulfillmentData `json:"fulfillmentData"`
}

// ProductVariantInput represents the data required to create or update a product variant.
type ProductVariantInput struct {
    SKU             string          `json:"sku"`
    Title           string          `json:"title"`
    Price           float64         `json:"price"`
    Inventory       int             `json:"inventory"`
    Options         []VariantOption `json:"options"`
    Dimensions      Dimensions      `json:"dimensions"`
    FulfillmentData FulfillmentData `json:"fulfillmentData"`
}