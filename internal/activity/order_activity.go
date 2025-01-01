package activity

import (
	"context"
	"time"
)

type OrderActivity struct {
	// In a real application, you would inject your repositories/services here
}

func NewOrderActivity() *OrderActivity {
	return &OrderActivity{}
}

type CreateOrderRequest struct {
	ID        string
	Status    string
	CreatedAt time.Time
}

type ProcessPaymentRequest struct {
	OrderID string
	Amount  float64
}

type UpdateInventoryRequest struct {
	OrderID    string
	ProductIDs []string
}

type CompensateOrderRequest struct {
	OrderID string
	Amount  float64
	Reason  string
	Steps   []string
}

func (a *OrderActivity) CreateOrder(ctx context.Context, req CreateOrderRequest) error {
	// Simulate database operation
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (a *OrderActivity) ProcessPayment(ctx context.Context, req ProcessPaymentRequest) error {
	// Simulate payment processing
	time.Sleep(200 * time.Millisecond)
	return nil
}

func (a *OrderActivity) UpdateInventory(ctx context.Context, req UpdateInventoryRequest) error {
	// Simulate inventory update
	time.Sleep(150 * time.Millisecond)
	return nil
}

func (a *OrderActivity) NotifyCustomer(ctx context.Context, orderID string) error {
	// Simulate notification sending
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (a *OrderActivity) CompensateOrder(ctx context.Context, req CompensateOrderRequest) error {
	// Step 1: If payment was processed, issue refund
	if contains(req.Steps, "payment") {
		if err := a.ProcessPayment(ctx, ProcessPaymentRequest{
			OrderID: req.OrderID,
			Amount:  -req.Amount, // Negative amount for refund
		}); err != nil {
			return err
		}
	}

	// Step 2: Update order status to cancelled
	if err := a.CreateOrder(ctx, CreateOrderRequest{
		ID:        req.OrderID,
		Status:    "cancelled",
		CreatedAt: time.Now(),
	}); err != nil {
		return err
	}

	// Step 3: Notify customer about cancellation/failure
	return a.NotifyCustomer(ctx, req.OrderID)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
