package models

import (
	"time"
)

// Cart represents a shopping cart.
type Cart struct {
	ID         string     `json:"id"`
	CustomerID string     `json:"customerId"`
	Items      []CartItem `json:"items"`
	TotalPrice float64    `json:"totalPrice"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	ExpiresAt  int64      `json:"expiresAt"`
}

// CartItem represents an item in a shopping cart.
type CartItem struct {
	ID              string          `json:"id"`
	ProductID       string          `json:"productId"`
	VariantID       string          `json:"variantId"`
	Quantity        int             `json:"quantity"`
	Price           float64         `json:"price"`
	FulfillmentData FulfillmentData `json:"fulfillmentData"`
}

// CartItemInput represents the data required to add or update a cart item.
type CartItemInput struct {
	ProductID string `json:"productId"`
	VariantID string `json:"variantId"`
	Quantity  int    `json:"quantity"`
}
