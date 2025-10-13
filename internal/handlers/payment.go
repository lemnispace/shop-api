package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/lemnispace/shop-api/internal/models"
	"github.com/lemnispace/shop-api/internal/services"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/webhook"
)

var (
	paymentService         services.PaymentService
	orderServiceForPayment services.OrderService
	fulfillmentService     services.FulfillmentService
)

// SetPaymentService sets the payment service instance
func SetPaymentService(service services.PaymentService) {
	paymentService = service
	log.Println("Payment service injected into handlers")
}

// SetOrderServiceForPayments sets the order service instance for payment handlers
func SetOrderServiceForPayments(service services.OrderService) {
	orderServiceForPayment = service
	log.Println("Order service injected into payment handlers")
}

// SetFulfillmentService sets the fulfillment service instance for payment handlers
func SetFulfillmentService(service services.FulfillmentService) {
	fulfillmentService = service
	log.Println("Fulfillment service injected into payment handlers")
}

// CreatePaymentIntent creates a Stripe PaymentIntent for an order
// POST /v1/orders/:orderId/payment-intent
func CreatePaymentIntent(c *gin.Context) {
	orderID := c.Param("orderId")

	// Get the order
	order, err := orderServiceForPayment.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		if err == services.ErrOrderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}
		log.Printf("[ERROR] Failed to get order %s: %v", orderID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get order"})
		return
	}

	// Validate order is in pending status
	if order.Status != models.OrderStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cannot create payment for order with status: %s", order.Status)})
		return
	}

	// Convert amount to cents (Stripe uses smallest currency unit)
	amountCents := int64(order.TotalPrice * 100)

	// Create payment intent
	metadata := map[string]string{
		"order_id":    order.ID,
		"customer_id": order.CustomerID,
	}

	pi, err := paymentService.CreatePaymentIntent(c.Request.Context(), amountCents, "usd", metadata)
	if err != nil {
		log.Printf("[ERROR] Failed to create payment intent for order %s: %v", orderID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment intent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"paymentIntent": gin.H{
			"id":           pi.ID,
			"clientSecret": pi.ClientSecret,
			"amount":       pi.Amount,
			"currency":     pi.Currency,
			"status":       pi.Status,
		},
		"order": order,
	})
}

// ConfirmPayment confirms a payment and updates order status
// POST /v1/orders/:orderId/confirm-payment
func ConfirmPayment(c *gin.Context) {
	orderID := c.Param("orderId")

	var input struct {
		PaymentIntentID string `json:"paymentIntentId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Get the order
	order, err := orderServiceForPayment.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		if err == services.ErrOrderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}
		log.Printf("[ERROR] Failed to get order %s: %v", orderID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get order"})
		return
	}

	// Get payment intent to verify it belongs to this order
	pi, err := paymentService.GetPaymentIntent(c.Request.Context(), input.PaymentIntentID)
	if err != nil {
		log.Printf("[ERROR] Failed to get payment intent %s: %v", input.PaymentIntentID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment intent"})
		return
	}

	// Verify payment intent matches order
	if pi.Metadata["order_id"] != orderID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment intent does not match order"})
		return
	}

	// Check if payment is successful
	if pi.Status == stripe.PaymentIntentStatusSucceeded {
		// Update order status to paid
		err = orderServiceForPayment.UpdateOrderStatus(c.Request.Context(), orderID, models.OrderStatusPaid)
		if err != nil {
			log.Printf("[ERROR] Failed to update order %s status to paid: %v", orderID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status"})
			return
		}

		log.Printf("Payment confirmed for order %s, payment intent: %s", orderID, input.PaymentIntentID)

		// Submit order to Printful for fulfillment (if fulfillment service is configured)
		if fulfillmentService != nil {
			// Fetch the updated order
			updatedOrder, err := orderServiceForPayment.GetOrder(c.Request.Context(), orderID)
			if err != nil {
				log.Printf("[ERROR] Failed to fetch order %s for Printful submission: %v", orderID, err)
			} else {
				// Submit to Printful asynchronously
				go func() {
					fulfillment, err := fulfillmentService.SubmitOrderToPrintful(c.Request.Context(), updatedOrder)
					if err != nil {
						log.Printf("[ERROR] Failed to submit order %s to Printful: %v", orderID, err)
					} else {
						log.Printf("[INFO] Successfully submitted order %s to Printful, fulfillment ID: %s", orderID, fulfillment.ID)
					}
				}()
			}
		} else {
			log.Printf("[WARNING] Fulfillment service not configured, skipping Printful submission for order %s", orderID)
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"order":   order,
			"payment": gin.H{
				"id":     pi.ID,
				"status": pi.Status,
				"amount": pi.Amount,
			},
		})
		return
	}

	// Payment not successful
	c.JSON(http.StatusBadRequest, gin.H{
		"success": false,
		"error":   "Payment not successful",
		"status":  pi.Status,
	})
}

// HandleStripeWebhook handles Stripe webhook events
// POST /v1/webhooks/stripe
func HandleStripeWebhook(c *gin.Context) {
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if webhookSecret == "" {
		log.Println("[WARNING] STRIPE_WEBHOOK_SECRET not set, skipping webhook signature verification")
	}

	// Read body
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("[ERROR] Failed to read webhook payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Verify webhook signature if secret is set
	var event stripe.Event
	if webhookSecret != "" {
		signatureHeader := c.GetHeader("Stripe-Signature")
		event, err = webhook.ConstructEvent(payload, signatureHeader, webhookSecret)
		if err != nil {
			log.Printf("[ERROR] Failed to verify webhook signature: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid signature"})
			return
		}
	} else {
		// Parse event without verification (development only)
		err = json.Unmarshal(payload, &event)
		if err != nil {
			log.Printf("[ERROR] Failed to parse webhook event: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event"})
			return
		}
	}

	// Handle the event
	switch event.Type {
	case "payment_intent.succeeded":
		var pi stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &pi)
		if err != nil {
			log.Printf("[ERROR] Failed to unmarshal payment intent: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment intent data"})
			return
		}

		// Get order ID from metadata
		orderID := pi.Metadata["order_id"]
		if orderID == "" {
			log.Printf("[WARNING] Payment intent %s has no order_id in metadata", pi.ID)
			c.JSON(http.StatusOK, gin.H{"received": true})
			return
		}

		// Update order status to paid
		err = orderServiceForPayment.UpdateOrderStatus(c.Request.Context(), orderID, models.OrderStatusPaid)
		if err != nil {
			log.Printf("[ERROR] Failed to update order %s status to paid: %v", orderID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order"})
			return
		}

		log.Printf("Webhook: Payment succeeded for order %s, payment intent: %s", orderID, pi.ID)

		// Submit order to Printful for fulfillment (if fulfillment service is configured)
		if fulfillmentService != nil {
			// Fetch the order
			order, err := orderServiceForPayment.GetOrder(c.Request.Context(), orderID)
			if err != nil {
				log.Printf("[ERROR] Failed to fetch order %s for Printful submission: %v", orderID, err)
			} else {
				// Submit to Printful asynchronously
				go func() {
					fulfillment, err := fulfillmentService.SubmitOrderToPrintful(c.Request.Context(), order)
					if err != nil {
						log.Printf("[ERROR] Failed to submit order %s to Printful: %v", orderID, err)
					} else {
						log.Printf("[INFO] Successfully submitted order %s to Printful via webhook, fulfillment ID: %s", orderID, fulfillment.ID)
					}
				}()
			}
		}

	case "payment_intent.payment_failed":
		var pi stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &pi)
		if err != nil {
			log.Printf("[ERROR] Failed to unmarshal payment intent: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment intent data"})
			return
		}

		orderID := pi.Metadata["order_id"]
		log.Printf("Webhook: Payment failed for order %s, payment intent: %s", orderID, pi.ID)

	case "payment_intent.canceled":
		var pi stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &pi)
		if err != nil {
			log.Printf("[ERROR] Failed to unmarshal payment intent: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment intent data"})
			return
		}

		orderID := pi.Metadata["order_id"]
		log.Printf("Webhook: Payment canceled for order %s, payment intent: %s", orderID, pi.ID)

	default:
		log.Printf("Webhook: Unhandled event type: %s", event.Type)
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}
