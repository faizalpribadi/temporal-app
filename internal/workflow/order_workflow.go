package workflow

import (
	"fmt"
	"time"

	"app/internal/activity"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type OrderSignal struct {
	Status string
}

type Order struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	Amount    float64   `json:"amount"`
	Products  []string  `json:"products"`
}

func OrderWorkflow(ctx workflow.Context, order Order) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("OrderWorkflow started", "OrderID", order.ID, "Amount", order.Amount, "Products", order.Products)

	// Activity options with retry policy
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Initialize activities
	activities := activity.NewOrderActivity()

	// Step 1: Create order in database
	logger.Info("Starting order creation", "OrderID", order.ID)
	createOrderErr := workflow.ExecuteActivity(ctx, activities.CreateOrder, activity.CreateOrderRequest{
		ID:        order.ID,
		Status:    order.Status,
		CreatedAt: order.CreatedAt,
	}).Get(ctx, nil)
	if createOrderErr != nil {
		logger.Error("Failed to create order", "OrderID", order.ID, "Error", createOrderErr)
		return createOrderErr
	}
	logger.Info("Order created successfully", "OrderID", order.ID)

	// Step 2: Process payment
	logger.Info("Starting payment processing", "OrderID", order.ID, "Amount", order.Amount)
	processPaymentErr := workflow.ExecuteActivity(ctx, activities.ProcessPayment, activity.ProcessPaymentRequest{
		OrderID: order.ID,
		Amount:  order.Amount,
	}).Get(ctx, nil)
	if processPaymentErr != nil {
		logger.Error("Payment processing failed", "OrderID", order.ID, "Error", processPaymentErr)

		// Compensate for payment failure
		compensateErr := workflow.ExecuteActivity(ctx, activities.CompensateOrder, activity.CompensateOrderRequest{
			OrderID: order.ID,
			Amount:  order.Amount,
			Reason:  "payment_failed",
			Steps:   []string{},
		}).Get(ctx, nil)
		if compensateErr != nil {
			logger.Error("Compensation failed", "OrderID", order.ID, "Error", compensateErr)
		}
		return fmt.Errorf("payment processing failed: %w", processPaymentErr)
	}
	logger.Info("Payment processed successfully", "OrderID", order.ID)

	// Step 3: Update inventory
	logger.Info("Starting inventory update", "OrderID", order.ID, "Products", order.Products)
	updateInventoryErr := workflow.ExecuteActivity(ctx, activities.UpdateInventory, activity.UpdateInventoryRequest{
		OrderID:    order.ID,
		ProductIDs: order.Products,
	}).Get(ctx, nil)
	if updateInventoryErr != nil {
		logger.Error("Inventory update failed", "OrderID", order.ID, "Error", updateInventoryErr)

		// Compensate for inventory update failure
		compensateErr := workflow.ExecuteActivity(ctx, activities.CompensateOrder, activity.CompensateOrderRequest{
			OrderID: order.ID,
			Amount:  order.Amount,
			Reason:  "inventory_failed",
			Steps:   []string{"payment"},
		}).Get(ctx, nil)
		if compensateErr != nil {
			logger.Error("Compensation failed", "OrderID", order.ID, "Error", compensateErr)
		}
		return fmt.Errorf("inventory update failed: %w", updateInventoryErr)
	}
	logger.Info("Inventory updated successfully", "OrderID", order.ID)

	// Channel for receiving signals
	signalChan := workflow.GetSignalChannel(ctx, "order-signal")
	var signal OrderSignal

	// Create a selector to handle the signal
	selector := workflow.NewSelector(ctx)
	selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &signal)
		logger.Info("Received signal", "Signal", signal.Status, "OrderID", order.ID)
	})

	// Wait for signals with a timeout
	timeout := workflow.NewTimer(ctx, 24*time.Hour)
	selector.AddFuture(timeout, func(f workflow.Future) {
		logger.Info("Workflow timed out after 24 hours", "OrderID", order.ID)
	})

	// Keep the workflow running until timeout or completion signal
	running := true
	for running {
		selector.Select(ctx)

		switch signal.Status {
		case "complete":
			running = false
			logger.Info("Processing order completion", "OrderID", order.ID)
			if err := workflow.ExecuteActivity(ctx, activities.NotifyCustomer, order.ID).Get(ctx, nil); err != nil {
				logger.Error("Failed to send completion notification", "OrderID", order.ID, "Error", err)
			} else {
				logger.Info("Completion notification sent successfully", "OrderID", order.ID)
			}

		case "cancel":
			running = false
			logger.Info("Processing order cancellation", "OrderID", order.ID)

			// Compensate for cancellation
			compensateErr := workflow.ExecuteActivity(ctx, activities.CompensateOrder, activity.CompensateOrderRequest{
				OrderID: order.ID,
				Amount:  order.Amount,
				Reason:  "user_cancelled",
				Steps:   []string{"payment", "inventory"},
			}).Get(ctx, nil)
			if compensateErr != nil {
				logger.Error("Cancellation compensation failed", "OrderID", order.ID, "Error", compensateErr)
			}
		}
	}

	return nil
}
