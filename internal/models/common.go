package models

// Dimensions represents the physical dimensions of a product or variant.
type Dimensions struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Depth  float64 `json:"depth"`
	Weight float64 `json:"weight"`
}

// FulfillmentData contains data required for fulfillment partners.
type FulfillmentData struct {
	PartnerID        string                 `json:"partnerId"`
	PartnerProductID string                 `json:"partnerProductId"`
	PartnerVariantID string                 `json:"partnerVariantId"`
	AdditionalData   map[string]interface{} `json:"additionalData"`
}

// VariantOption represents an option for a product variant.
type VariantOption struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Image represents an image associated with a product.
type Image struct {
	ID      string `json:"id"`
	URL     string `json:"url"`
	AltText string `json:"altText"`
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
