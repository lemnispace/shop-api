package models

import "time"

// PrintfulProduct represents a product from the Printful catalog
type PrintfulProduct struct {
	ID             int              `json:"id"`
	ExternalID     string           `json:"external_id,omitempty"`
	Name           string           `json:"title"`
	Variants       int              `json:"variants"`
	Synced         int              `json:"synced"`
	Thumbnail      string           `json:"thumbnail_url"`
	IsSyncable     bool             `json:"is_syncable"`
	Description    string           `json:"description"`
	Category       string           `json:"category"`
	MockupImages   []PrintfulImage  `json:"mockup_images,omitempty"`
	FileSpec       PrintfulFileSpec `json:"file_spec,omitempty"`
	Options        []PrintfulOption `json:"options,omitempty"`
	Dimensions     Dimensions       `json:"dimensions,omitempty"`
	IsDiscontinued bool             `json:"is_discontinued"`
	AvgFulfillment int              `json:"avg_fulfillment_time,omitempty"`
}

// PrintfulVariant represents a variant of a Printful product
type PrintfulVariant struct {
	ID           int                   `json:"id"`
	ProductID    int                   `json:"product_id"`
	Name         string                `json:"name"`
	Size         string                `json:"size"`
	Color        string                `json:"color"`
	ColorCode    string                `json:"color_code"`
	Image        string                `json:"image"`
	Price        string                `json:"price"`
	InStock      bool                  `json:"in_stock"`
	Availability interface{}           `json:"availability_status,omitempty"` // Can be string or array of objects
	Options      []PrintfulOptionValue `json:"options,omitempty"`
	Dimensions   Dimensions            `json:"dimensions,omitempty"`
}

// PrintfulImage represents an image from Printful
type PrintfulImage struct {
	ID       int    `json:"id,omitempty"`
	URL      string `json:"url"`
	Type     string `json:"type,omitempty"`
	Hash     string `json:"hash,omitempty"`
	Filename string `json:"filename,omitempty"`
	Position int    `json:"position,omitempty"`
}

// PrintfulFileSpec represents file specifications for a product
type PrintfulFileSpec struct {
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Format    string `json:"format"`
	DPI       int    `json:"dpi"`
	ColorMode string `json:"color_mode,omitempty"`
}

// PrintfulOption represents a product option (size, color, etc.)
type PrintfulOption struct {
	ID     string      `json:"id"`
	Title  string      `json:"title"`
	Type   string      `json:"type"`
	Values interface{} `json:"values,omitempty"` // Can be []string or map[string]string depending on API response
}

// PrintfulOptionValue represents a selected option value
type PrintfulOptionValue struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// PrintfulOrderRequest represents a request to create an order with Printful
type PrintfulOrderRequest struct {
	ExternalID  string               `json:"external_id"`
	Recipient   PrintfulRecipient    `json:"recipient"`
	Items       []PrintfulOrderItem  `json:"items"`
	RetailCosts *PrintfulRetailCosts `json:"retail_costs,omitempty"`
	GiftMessage string               `json:"gift_message,omitempty"`
	PackingSlip *PrintfulPackingSlip `json:"packing_slip,omitempty"`
}

// PrintfulRecipient represents the shipping recipient
type PrintfulRecipient struct {
	Name        string `json:"name"`
	Company     string `json:"company,omitempty"`
	Address1    string `json:"address1"`
	Address2    string `json:"address2,omitempty"`
	City        string `json:"city"`
	StateCode   string `json:"state_code"`
	StateName   string `json:"state_name,omitempty"`
	CountryCode string `json:"country_code"`
	CountryName string `json:"country_name,omitempty"`
	Zip         string `json:"zip"`
	Phone       string `json:"phone,omitempty"`
	Email       string `json:"email,omitempty"`
}

// PrintfulOrderItem represents an item in a Printful order
type PrintfulOrderItem struct {
	ID                int                   `json:"id,omitempty"`
	ExternalID        string                `json:"external_id,omitempty"`
	VariantID         int                   `json:"variant_id,omitempty"`
	SyncVariantID     int                   `json:"sync_variant_id,omitempty"`
	ExternalVariantID string                `json:"external_variant_id,omitempty"`
	Quantity          int                   `json:"quantity"`
	Price             string                `json:"price,omitempty"`
	RetailPrice       string                `json:"retail_price,omitempty"`
	Name              string                `json:"name,omitempty"`
	Product           *PrintfulProduct      `json:"product,omitempty"`
	Files             []PrintfulOrderFile   `json:"files,omitempty"`
	Options           []PrintfulOptionValue `json:"options,omitempty"`
	SKU               string                `json:"sku,omitempty"`
}

// PrintfulOrderFile represents a file for customization
type PrintfulOrderFile struct {
	ID        int                  `json:"id,omitempty"`
	Type      string               `json:"type,omitempty"`
	URL       string               `json:"url"`
	Options   []PrintfulFileOption `json:"options,omitempty"`
	Hash      string               `json:"hash,omitempty"`
	Filename  string               `json:"filename,omitempty"`
	MIMEType  string               `json:"mime_type,omitempty"`
	Size      int                  `json:"size,omitempty"`
	Width     int                  `json:"width,omitempty"`
	Height    int                  `json:"height,omitempty"`
	DPI       int                  `json:"dpi,omitempty"`
	Status    string               `json:"status,omitempty"`
	Created   int64                `json:"created,omitempty"`
	Thumbnail string               `json:"thumbnail_url,omitempty"`
	Preview   string               `json:"preview_url,omitempty"`
	Visible   bool                 `json:"visible,omitempty"`
}

// PrintfulFileOption represents options for file placement
type PrintfulFileOption struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// PrintfulRetailCosts represents retail pricing
type PrintfulRetailCosts struct {
	Currency string `json:"currency"`
	Subtotal string `json:"subtotal,omitempty"`
	Discount string `json:"discount,omitempty"`
	Shipping string `json:"shipping,omitempty"`
	Tax      string `json:"tax,omitempty"`
}

// PrintfulPackingSlip represents a packing slip
type PrintfulPackingSlip struct {
	Email    string `json:"email,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Message  string `json:"message,omitempty"`
	LogoURL  string `json:"logo_url,omitempty"`
	StoreURL string `json:"store_name,omitempty"`
}

// PrintfulOrder represents an order from Printful
type PrintfulOrder struct {
	ID           int                  `json:"id"`
	ExternalID   string               `json:"external_id"`
	Status       string               `json:"status"`
	Shipping     string               `json:"shipping"`
	Created      int64                `json:"created"`
	Updated      int64                `json:"updated"`
	Recipient    PrintfulRecipient    `json:"recipient"`
	Items        []PrintfulOrderItem  `json:"items"`
	Costs        *PrintfulCosts       `json:"costs,omitempty"`
	RetailCosts  *PrintfulRetailCosts `json:"retail_costs,omitempty"`
	ShipmentInfo *PrintfulShipment    `json:"shipments,omitempty"`
	GiftMessage  string               `json:"gift_message,omitempty"`
	PackingSlip  *PrintfulPackingSlip `json:"packing_slip,omitempty"`
}

// PrintfulCosts represents costs from Printful
type PrintfulCosts struct {
	Currency     string `json:"currency"`
	Subtotal     string `json:"subtotal"`
	Discount     string `json:"discount"`
	Shipping     string `json:"shipping"`
	Digitization string `json:"digitization,omitempty"`
	Tax          string `json:"tax"`
	Vat          string `json:"vat,omitempty"`
	Total        string `json:"total"`
}

// PrintfulShipment represents shipment information
type PrintfulShipment struct {
	ID             int    `json:"id"`
	Carrier        string `json:"carrier"`
	Service        string `json:"service"`
	TrackingNumber string `json:"tracking_number"`
	TrackingURL    string `json:"tracking_url"`
	Created        int64  `json:"created"`
	ShipDate       string `json:"ship_date,omitempty"`
	Shipped        int64  `json:"shipped_at,omitempty"`
	ReShipped      int64  `json:"reshipment_available,omitempty"`
}

// PrintfulSyncJob represents a catalog sync job
type PrintfulSyncJob struct {
	ID             string     `json:"id"`
	Type           string     `json:"type"`
	Status         string     `json:"status"`
	Progress       int        `json:"progress"`
	ItemsProcessed int        `json:"itemsProcessed"`
	ItemsTotal     int        `json:"itemsTotal"`
	StartedAt      time.Time  `json:"startedAt"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
	Error          string     `json:"error,omitempty"`
}

// PrintfulProductImportRequest represents a request to import a specific product
type PrintfulProductImportRequest struct {
	PrintfulProductID string   `json:"printfulProductId"`
	VariantIDs        []string `json:"variantIds,omitempty"`
	Title             string   `json:"title,omitempty"`
	Description       string   `json:"description,omitempty"`
	MarkupPercentage  float64  `json:"markupPercentage"`
}

// PrintfulAPIResponse is a generic wrapper for Printful API responses
type PrintfulAPIResponse struct {
	Code   int            `json:"code"`
	Result interface{}    `json:"result"`
	Extra  interface{}    `json:"extra,omitempty"`
	Error  *PrintfulError `json:"error,omitempty"`
}

// PrintfulError represents an error from the Printful API
type PrintfulError struct {
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// PrintfulListResponse represents a list response from Printful
type PrintfulListResponse struct {
	Items      []interface{} `json:"items"`
	NextCursor string        `json:"next_cursor,omitempty"`
	HasMore    bool          `json:"has_more"`
}
