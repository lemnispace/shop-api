package models

import (
	"time"
)

// DiscountType represents the type of discount.
type DiscountType string

const (
	DiscountTypePercentage   DiscountType = "percentage"
	DiscountTypeFixedAmount  DiscountType = "fixed_amount"
	DiscountTypeFreeShipping DiscountType = "free_shipping"
)

// AppliesTo represents what the discount applies to.
type AppliesTo string

const (
	AppliesToAll                 AppliesTo = "all"
	AppliesToSpecificProducts    AppliesTo = "specific_products"
	AppliesToSpecificCollections AppliesTo = "specific_collections"
)

// Discount represents a discount code.
type Discount struct {
	ID                    string       `json:"id"`
	Code                  string       `json:"code"`
	Type                  DiscountType `json:"type"`
	Value                 float64      `json:"value"`
	MinimumPurchaseAmount float64      `json:"minimumPurchaseAmount"`
	AppliesTo             AppliesTo    `json:"appliesTo"`
	TargetSelection       []string     `json:"targetSelection"`
	StartsAt              time.Time    `json:"startsAt"`
	EndsAt                time.Time    `json:"endsAt"`
	CreatedAt             time.Time    `json:"createdAt"`
	UpdatedAt             time.Time    `json:"updatedAt"`
}

// DiscountInput represents the data required to create or update a discount.
type DiscountInput struct {
	Code                  string       `json:"code"`
	Type                  DiscountType `json:"type"`
	Value                 float64      `json:"value"`
	MinimumPurchaseAmount float64      `json:"minimumPurchaseAmount"`
	AppliesTo             AppliesTo    `json:"appliesTo"`
	TargetSelection       []string     `json:"targetSelection"`
	StartsAt              time.Time    `json:"startsAt"`
	EndsAt                time.Time    `json:"endsAt"`
}
