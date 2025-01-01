package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"app/internal/activity"
	"app/internal/handler"
	"app/internal/workflow"

	"github.com/gin-gonic/gin"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	// slog json
	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}),
	)

	// Create Temporal client
	temporalClient, err := client.NewLazyClient(client.Options{
		Logger: logger,
	})
	if err != nil {
		logger.Error("Unable to create Temporal client", "error", err.Error())
	}
	defer temporalClient.Close()

	// Create Temporal worker
	w := worker.New(temporalClient, "order-queue", worker.Options{})
	w.RegisterWorkflow(workflow.OrderWorkflow)

	// Register activities
	activities := activity.NewOrderActivity()
	w.RegisterActivity(activities.CreateOrder)
	w.RegisterActivity(activities.ProcessPayment)
	w.RegisterActivity(activities.UpdateInventory)
	w.RegisterActivity(activities.NotifyCustomer)
	w.RegisterActivity(activities.CompensateOrder)

	// Start worker (non-blocking)
	if err := w.Start(); err != nil {
		logger.Error("Unable to start worker", "error", err.Error())
		os.Exit(1)
	}

	// Create Gin router
	router := gin.Default()

	// Create order handler
	orderHandler := handler.NewOrderHandler(temporalClient)

	// Register routes
	router.POST("/order", orderHandler.CreateOrder)
	router.GET("/order/:orderId", orderHandler.GetOrder)
	router.PUT("/order/:orderId/status", orderHandler.UpdateOrderStatus)

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", "error", err.Error())
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Shutdown server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err.Error())
		os.Exit(1)
	}
}
