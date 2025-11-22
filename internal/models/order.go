package models

import (
	"time"
)

// OrderStatus represents the possible statuses of an order.
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusFulfilled OrderStatus = "fulfilled"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// Order represents a customer's order.
type Order struct {
	ID                   string        `json:"id"`
	CustomerID           string        `json:"customerId"`
	Items                []CartItem    `json:"items"`
	Subtotal             int64         `json:"subtotal"`   // Amount in cents
	Tax                  int64         `json:"tax"`        // Amount in cents
	Shipping             int64         `json:"shipping"`   // Amount in cents
	TotalPrice           int64         `json:"totalPrice"` // Amount in cents
	Status               OrderStatus   `json:"status"`
	ShippingAddress      Address       `json:"shippingAddress"`
	BillingAddress       Address       `json:"billingAddress"`
	ShippingMethod       string        `json:"shippingMethod"`
	PaymentMethod        string        `json:"paymentMethod"`
	FulfillmentPartnerID string        `json:"fulfillmentPartnerId"`
	Fulfillments         []Fulfillment `json:"fulfillments"`
	CreatedAt            time.Time     `json:"createdAt"`
	UpdatedAt            time.Time     `json:"updatedAt"`
}

// OrderInput represents the data required to create a new order.
type OrderInput struct {
	CartID          string  `json:"cartId"`
	CustomerID      string  `json:"customerId"`
	ShippingAddress Address `json:"shippingAddress"`
	BillingAddress  Address `json:"billingAddress"`
	ShippingMethod  string  `json:"shippingMethod"`
	PaymentMethod   string  `json:"paymentMethod"`
}
