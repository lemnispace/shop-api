package models

import (
	"time"
)

// Customer represents a customer in the e-commerce platform.
type Customer struct {
	ID               string    `json:"id"`
	Email            string    `json:"email"`
	FirstName        string    `json:"firstName"`
	LastName         string    `json:"lastName"`
	Phone            string    `json:"phone"`
	AcceptsMarketing bool      `json:"acceptsMarketing"`
	Tags             []string  `json:"tags"`
	DefaultAddress   Address   `json:"defaultAddress"`
	Addresses        []Address `json:"addresses"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// CustomerInput represents the data required to create or update a customer.
type CustomerInput struct {
	Email            string  `json:"email"`
	Password         string  `json:"password"`
	FirstName        string  `json:"firstName"`
	LastName         string  `json:"lastName"`
	Phone            string  `json:"phone"`
	AcceptsMarketing bool    `json:"acceptsMarketing"`
	DefaultAddress   Address `json:"defaultAddress"`
}

// CustomerLoginInput represents the data required for customer authentication.
type CustomerLoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// CustomerLoginResponse represents the authentication token returned upon successful login.
type CustomerLoginResponse struct {
	Token string `json:"token"`
}
