package models

import (
	"time"
)

// Cart represents a shopping cart.
type Cart struct {
	ID                string     `json:"id"`
	CustomerID        string     `json:"customerId"`
	Items             []CartItem `json:"items"`
	Subtotal          int64      `json:"subtotal"`          // Amount in cents
	EstimatedTax      int64      `json:"estimatedTax"`      // Amount in cents
	EstimatedShipping int64      `json:"estimatedShipping"` // Amount in cents
	TotalPrice        int64      `json:"totalPrice"`        // Amount in cents
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
	ExpiresAt         time.Time  `json:"expiresAt"`
}

// CartItem represents an item in a shopping cart.
type CartItem struct {
	ID                string                 `json:"id"`
	ProductID         string                 `json:"productId"`
	VariantID         string                 `json:"variantId"`
	Quantity          int                    `json:"quantity"`
	Price             int64                  `json:"price"` // Price per unit in cents
	CustomizationData map[string]interface{} `json:"customizationData,omitempty"`
	Product           *CartItemProduct       `json:"product,omitempty"`
	Variant           *CartItemVariant       `json:"variant,omitempty"`
	FulfillmentData   FulfillmentData        `json:"fulfillmentData,omitempty"`
}

// CartItemProduct represents the product information in a cart item
type CartItemProduct struct {
	Title string `json:"title"`
	Image string `json:"image"`
}

// CartItemVariant represents the variant information in a cart item
type CartItemVariant struct {
	Title string `json:"title"`
}

// CartItemInput represents the data required to add or update a cart item.
type CartItemInput struct {
	ProductID         string                 `json:"productId"`
	VariantID         string                 `json:"variantId"`
	Quantity          int                    `json:"quantity"`
	CustomizationData map[string]interface{} `json:"customizationData,omitempty"`
}

// CheckoutResponse represents the response when requesting a checkout URL
type CheckoutResponse struct {
	CheckoutURL string `json:"checkoutUrl"`
}
