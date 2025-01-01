package handler

import (
	"fmt"
	"net/http"
	"time"

	"app/internal/workflow"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
)

type CreateOrderRequest struct {
	Amount   float64   `json:"amount" binding:"required"`
	Products []string  `json:"products" binding:"required"`
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type Order struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	Amount    float64   `json:"amount"`
	Products  []string  `json:"products"`
}

type OrderHandler struct {
	temporalClient client.Client
}

func NewOrderHandler(temporalClient client.Client) *OrderHandler {
	return &OrderHandler{
		temporalClient: temporalClient,
	}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order := workflow.Order{
		ID:        uuid.New().String(),
		Status:    "pending",
		CreatedAt: time.Now(),
		Amount:    req.Amount,
		Products:  req.Products,
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:        "order-" + order.ID,
		TaskQueue: "order-queue",
	}

	_, err := h.temporalClient.ExecuteWorkflow(c, workflowOptions, "OrderWorkflow", order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, order)
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("orderId")

	// For simplicity, we're returning a mock order
	// In a real application, you would fetch the order details from a database
	order := Order{
		ID:        orderID,
		Status:    "pending",
		CreatedAt: time.Now(),
		Amount:    0,
		Products:  []string{},
	}

	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	orderID := c.Param("orderId")
	var req UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate status
	switch req.Status {
	case "complete", "cancel":
		// Valid statuses
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid status: %s. Must be 'complete' or 'cancel'", req.Status)})
		return
	}

	workflowID := "order-" + orderID

	// Send signal to the workflow
	signal := workflow.OrderSignal{
		Status: req.Status,
	}

	err := h.temporalClient.SignalWorkflow(c, workflowID, "", "order-signal", signal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Order status updated to %s successfully", req.Status),
		"orderId": orderID,
		"status": req.Status,
	})
}
