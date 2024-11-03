package models

import (
	"time"
)

// Collection represents a grouping of products.
type Collection struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Products    []Product `json:"products"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// CollectionInput represents the data required to create or update a collection.
type CollectionInput struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	ProductIDs  []string `json:"productIds"`
}
