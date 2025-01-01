# Temporal Order Service

A robust order processing service built with Temporal and Gin, demonstrating the implementation of the Saga pattern and signal-based workflow control.

## Features

- **Order Management**
  - Create new orders
  - Process payments
  - Update inventory
  - Handle order completion and cancellation
  - Automatic compensation for failures

- **Saga Pattern Implementation**
  - Distributed transaction management
  - Automatic compensation on failures
  - Step-by-step transaction tracking
  - Idempotent operations

- **Signal-based Workflow Control**
  - Dynamic order status updates
  - Support for order completion
  - Support for order cancellation
  - 24-hour timeout handling

## Architecture

### Components

1. **Workflow (`internal/workflow/order_workflow.go`)**
   - Orchestrates the entire order process
   - Handles compensation logic
   - Processes signals for status updates
   - Implements timeout mechanism

2. **Activities (`internal/activity/order_activity.go`)**
   - `CreateOrder`: Manages order records
   - `ProcessPayment`: Handles payment transactions
   - `UpdateInventory`: Manages product inventory
   - `NotifyCustomer`: Sends customer notifications
   - `CompensateOrder`: Handles failure compensation

3. **HTTP Handler (`internal/handler/order_handler.go`)**
   - RESTful API endpoints
   - Request validation
   - Workflow interaction

## API Endpoints

### Create Order
```bash
curl -X POST http://localhost:8080/order \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 100.50,
    "products": ["product1", "product2"]
  }'
```

### Get Order Status
```bash
curl http://localhost:8080/order/{orderId}
```

### Update Order Status
```bash
curl -X PUT http://localhost:8080/order/{orderId}/status \
  -H "Content-Type: application/json" \
  -d '{"status": "complete"}'
```

## Compensation Strategy

The service implements a sophisticated compensation strategy through the `CompensateOrder` activity:

1. **Payment Failure**
   - Cancel order
   - No payment refund needed
   - Notify customer

2. **Inventory Failure**
   - Refund payment
   - Cancel order
   - Notify customer

3. **User Cancellation**
   - Refund payment
   - Revert inventory
   - Update order status
   - Notify customer

## Setup and Running

### Prerequisites
- Go 1.19 or later
- Temporal server running locally
- Git

### Installation
1. Clone the repository
```bash
git clone <repository-url>
cd app
```

2. Install dependencies
```bash
go mod download
```

3. Start the service
```bash
go run cmd/main.go
```

### Development Environment
- Temporal Web UI: http://localhost:8088
- API Server: http://localhost:8080

## Logging

The service uses structured JSON logging with different levels:
- INFO: Normal operation events
- ERROR: Operation failures and compensation events
- DEBUG: Detailed operation tracking

## Error Handling

1. **Retry Policy**
   - Initial interval: 1 second
   - Backoff coefficient: 2.0
   - Maximum interval: 1 minute
   - Maximum attempts: 3

2. **Compensation**
   - Centralized compensation logic
   - Step-based compensation tracking
   - Failure reason tracking

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a new Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
