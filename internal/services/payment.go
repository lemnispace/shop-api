package services

import (
	"context"
	"fmt"
	"log"

	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/paymentintent"
	"github.com/stripe/stripe-go/v78/refund"
)

// PaymentService defines the interface for payment operations
type PaymentService interface {
	CreatePaymentIntent(ctx context.Context, amount int64, currency string, metadata map[string]string) (*stripe.PaymentIntent, error)
	ConfirmPayment(ctx context.Context, paymentIntentID string) (*stripe.PaymentIntent, error)
	RefundPayment(ctx context.Context, paymentIntentID string, amount int64) (*stripe.Refund, error)
	GetPaymentIntent(ctx context.Context, paymentIntentID string) (*stripe.PaymentIntent, error)
}

// StripePaymentService implements PaymentService using Stripe
type StripePaymentService struct {
	apiKey string
}

// NewPaymentService creates a new Stripe-backed payment service
func NewPaymentService(apiKey string) *StripePaymentService {
	stripe.Key = apiKey
	log.Printf("Initializing Stripe Payment Service")
	return &StripePaymentService{
		apiKey: apiKey,
	}
}

// CreatePaymentIntent creates a new Stripe PaymentIntent
func (s *StripePaymentService) CreatePaymentIntent(ctx context.Context, amount int64, currency string, metadata map[string]string) (*stripe.PaymentIntent, error) {
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amount),
		Currency: stripe.String(currency),
	}

	// Add metadata
	if metadata != nil {
		for key, value := range metadata {
			params.AddMetadata(key, value)
		}
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		log.Printf("[ERROR] Failed to create payment intent: %v", err)
		return nil, fmt.Errorf("failed to create payment intent: %w", err)
	}

	log.Printf("Created payment intent: %s for amount: %d %s", pi.ID, amount, currency)
	return pi, nil
}

// ConfirmPayment confirms a payment intent
func (s *StripePaymentService) ConfirmPayment(ctx context.Context, paymentIntentID string) (*stripe.PaymentIntent, error) {
	params := &stripe.PaymentIntentConfirmParams{}

	pi, err := paymentintent.Confirm(paymentIntentID, params)
	if err != nil {
		log.Printf("[ERROR] Failed to confirm payment intent %s: %v", paymentIntentID, err)
		return nil, fmt.Errorf("failed to confirm payment: %w", err)
	}

	log.Printf("Confirmed payment intent: %s, status: %s", pi.ID, pi.Status)
	return pi, nil
}

// RefundPayment creates a refund for a payment
func (s *StripePaymentService) RefundPayment(ctx context.Context, paymentIntentID string, amount int64) (*stripe.Refund, error) {
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(paymentIntentID),
	}

	// If amount is specified, do partial refund
	if amount > 0 {
		params.Amount = stripe.Int64(amount)
	}

	r, err := refund.New(params)
	if err != nil {
		log.Printf("[ERROR] Failed to create refund for payment intent %s: %v", paymentIntentID, err)
		return nil, fmt.Errorf("failed to create refund: %w", err)
	}

	log.Printf("Created refund: %s for payment intent: %s, amount: %d", r.ID, paymentIntentID, r.Amount)
	return r, nil
}

// GetPaymentIntent retrieves a payment intent by ID
func (s *StripePaymentService) GetPaymentIntent(ctx context.Context, paymentIntentID string) (*stripe.PaymentIntent, error) {
	pi, err := paymentintent.Get(paymentIntentID, nil)
	if err != nil {
		log.Printf("[ERROR] Failed to get payment intent %s: %v", paymentIntentID, err)
		return nil, fmt.Errorf("failed to get payment intent: %w", err)
	}

	return pi, nil
}
