package models

import (
	"time"
)

// FulfillmentStatus represents the possible statuses of a fulfillment.
type FulfillmentStatus string

const (
	FulfillmentStatusPending    FulfillmentStatus = "pending"
	FulfillmentStatusProcessing FulfillmentStatus = "processing"
	FulfillmentStatusShipped    FulfillmentStatus = "shipped"
	FulfillmentStatusDelivered  FulfillmentStatus = "delivered"
	FulfillmentStatusCancelled  FulfillmentStatus = "cancelled"
)

// Fulfillment represents a fulfillment record for an order.
type Fulfillment struct {
	ID             string            `json:"id"`
	OrderID        string            `json:"orderId"`
	Status         FulfillmentStatus `json:"status"`
	TrackingNumber string            `json:"trackingNumber"`
	TrackingURL    string            `json:"trackingUrl"`
	Items          []FulfillmentItem `json:"items"`
	PartnerID      string            `json:"partnerId"`
	PartnerOrderID string            `json:"partnerOrderId"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
}

// FulfillmentInput represents the data required to create a new fulfillment.
type FulfillmentInput struct {
	OrderID   string                 `json:"orderId"`
	Items     []FulfillmentItemInput `json:"items"`
	PartnerID string                 `json:"partnerId"`
}

// FulfillmentItem represents an item within a fulfillment.
type FulfillmentItem struct {
	ID          string `json:"id"`
	OrderItemID string `json:"orderItemId"`
	Quantity    int    `json:"quantity"`
}

// FulfillmentItemInput represents the data required to add an item to a fulfillment.
type FulfillmentItemInput struct {
	OrderItemID string `json:"orderItemId"`
	Quantity    int    `json:"quantity"`
}
