# E-commerce Backend Example

A complete e-commerce backend demonstrating order processing workflows, saga pattern, payment integration with circuit breakers, and event-driven architecture.

## Overview

This example demonstrates:
- Complex entity relationships for e-commerce
- Order processing workflow with saga pattern (automatic rollback)
- Payment gateway integration with circuit breaker
- Shipping provider integration
- Inventory management with reservations
- Event-driven order notifications
- Multi-step workflows with compensation

## File Structure

```
03-ecommerce/
├── ecommerce.cai    # Main DSL file
├── README.md           # This file
└── test.sh             # Sample curl commands for testing
```

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         E-commerce Backend                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐             │
│  │ Customer │──▶│   Cart   │──▶│  Order   │──▶│ Payment  │             │
│  └──────────┘   └──────────┘   └──────────┘   └──────────┘             │
│       │              │              │                                    │
│       ▼              ▼              ▼                                    │
│  ┌──────────┐   ┌──────────┐   ┌───────────┐                            │
│  │ Address  │   │ CartItem │   │ OrderItem │                            │
│  └──────────┘   └──────────┘   └───────────┘                            │
│                      │                                                   │
│                      ▼                                                   │
│  ┌──────────┐   ┌──────────┐   ┌───────────────────┐                    │
│  │ Category │◀──│ Product  │──▶│ Inventory         │                    │
│  └──────────┘   └──────────┘   │ InventoryReserve  │                    │
│                                └───────────────────┘                    │
│                                                                          │
│  ═══════════════════════════════════════════════════════════════════    │
│                            Workflows                                     │
│  ═══════════════════════════════════════════════════════════════════    │
│                                                                          │
│  ┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐       │
│  │ OrderProcessing │──▶│ OrderFulfillment│──▶│ ProcessRefund   │       │
│  │   (Saga)        │   │                 │   │                 │       │
│  └─────────────────┘   └─────────────────┘   └─────────────────┘       │
│         │                      │                      │                  │
│         └──────────────────────┼──────────────────────┘                  │
│                                ▼                                         │
│  ═══════════════════════════════════════════════════════════════════    │
│                         Integrations                                     │
│  ═══════════════════════════════════════════════════════════════════    │
│                                                                          │
│  ┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐       │
│  │ PaymentGateway  │   │ ShippingProvider│   │  EmailService   │       │
│  │ (Stripe)        │   │                 │   │                 │       │
│  │ • Circuit Break │   │ • Circuit Break │   │ • Retry         │       │
│  │ • Retry         │   │ • Retry         │   │                 │       │
│  └─────────────────┘   └─────────────────┘   └─────────────────┘       │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## Entities

### Core Entities

| Entity | Description |
|--------|-------------|
| `Customer` | User accounts with loyalty tiers |
| `Address` | Shipping/billing addresses |
| `Category` | Product categories (hierarchical) |
| `Product` | Product catalog with inventory tracking |
| `Cart` / `CartItem` | Shopping cart |
| `Order` / `OrderItem` | Orders with status workflow |
| `Payment` | Payment transactions |
| `Inventory` | Stock levels and reservations |

### Order Status Workflow

```
pending → confirmed → processing → shipped → delivered
    │         │            │
    └─────────┴────────────┴──▶ cancelled
                               refunded
```

### Payment Status Workflow

```
pending → authorized → captured → partially_refunded → refunded
    │
    └──▶ failed
```

## Workflows

### 1. Order Processing (Saga Pattern)

The order processing workflow uses the **saga pattern** for automatic rollback on failure:

```
┌───────────────────────────────────────────────────────────────────────┐
│                    OrderProcessing Workflow                           │
├───────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐              │
│  │ 1. Validate  │──▶│ 2. Reserve   │──▶│ 3. Process   │              │
│  │    Order     │   │   Inventory  │   │   Payment    │              │
│  └──────────────┘   └──────────────┘   └──────────────┘              │
│         │                  │                  │                       │
│         │                  │                  │                       │
│         │          ┌───────┴───────┐  ┌──────┴──────┐                │
│         │          │ Compensation: │  │ Compensation:│                │
│         │          │ Release       │  │ Refund       │                │
│         │          │ Inventory     │  │ Payment      │                │
│         │          └───────────────┘  └──────────────┘                │
│         │                                                             │
│         ▼                                                             │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐              │
│  │ 4. Confirm   │──▶│ 5. Send      │──▶│ 6. Notify    │              │
│  │    Order     │   │   Email      │   │  Fulfillment │              │
│  └──────────────┘   └──────────────┘   └──────────────┘              │
│                          │                                            │
│                   (non-critical,                                      │
│                    no rollback)                                       │
│                                                                       │
└───────────────────────────────────────────────────────────────────────┘
```

**Compensation (Rollback):**
- If payment fails → Release inventory reservations
- If fulfillment fails → Refund payment and release inventory

### 2. Order Fulfillment

Handles picking, packing, and shipping:

1. Commit inventory reservation
2. Calculate shipping rates
3. Create shipment label
4. Update tracking information
5. Send shipping notification

### 3. Refund Processing

Handles refund requests:

1. Validate refund eligibility
2. Process refund with payment provider
3. Update order status
4. Restore inventory (if items returned)
5. Notify customer

## Integrations

### Payment Gateway (Stripe)

| Feature | Configuration |
|---------|---------------|
| Timeout | 30 seconds |
| Retry | 3 times with exponential backoff |
| Circuit Breaker | Opens after 5 failures in 1 minute |

**Operations:**
- `create_payment_intent` - Initiate payment
- `capture_payment` - Capture authorized payment
- `create_refund` - Process refund

### Shipping Provider

| Feature | Configuration |
|---------|---------------|
| Timeout | 15 seconds |
| Retry | 2 times with exponential backoff |
| Circuit Breaker | Opens after 10 failures in 5 minutes |

**Operations:**
- `calculate_rates` - Get shipping options
- `create_shipment` - Create label
- `track_shipment` - Track delivery

### Email Service

| Feature | Configuration |
|---------|---------------|
| Timeout | 10 seconds |
| Retry | 3 times with linear backoff |

## API Endpoints

### Products

| Method | Path | Description |
|--------|------|-------------|
| GET | `/products` | List products (with filters) |
| GET | `/products/{slug}` | Get product details |
| POST | `/products` | Create product (admin) |
| PUT | `/products/{id}` | Update product (admin) |

### Cart

| Method | Path | Description |
|--------|------|-------------|
| GET | `/cart` | Get current cart |
| POST | `/cart/items` | Add item to cart |
| PUT | `/cart/items/{id}` | Update quantity |
| DELETE | `/cart/items/{id}` | Remove item |
| POST | `/cart/coupon` | Apply coupon |
| DELETE | `/cart/coupon` | Remove coupon |

### Orders

| Method | Path | Description |
|--------|------|-------------|
| POST | `/orders` | Create order (checkout) |
| GET | `/orders` | List customer orders |
| GET | `/orders/{number}` | Get order details |
| POST | `/orders/{number}/cancel` | Cancel order |
| POST | `/orders/{number}/refund` | Request refund |

### Admin

| Method | Path | Description |
|--------|------|-------------|
| GET | `/admin/orders` | List all orders |
| PUT | `/admin/orders/{id}/status` | Update status |

## Step-by-Step Instructions

### 1. Generate the API

```bash
codeai generate examples/03-ecommerce/ecommerce.cai
```

### 2. Configure Environment

```bash
# Database
export DATABASE_URL="postgres://localhost:5432/ecommerce"
export REDIS_URL="redis://localhost:6379"

# JWT
export JWT_ISSUER="ecommerce-api"
export JWT_SECRET="your-secret-key"

# Stripe
export STRIPE_API_URL="https://api.stripe.com/v1"
export STRIPE_SECRET_KEY="sk_test_..."

# Shipping
export SHIPPING_API_URL="https://api.shipengine.com/v1"
export SHIPPING_API_KEY="your-shipping-key"

# Email
export EMAIL_API_URL="https://api.sendgrid.com/v3"
export EMAIL_API_KEY="your-email-key"
```

### 3. Run Migrations

```bash
codeai migrate up
```

### 4. Start the Server

```bash
codeai run
```

## Sample Requests

### Create a Product (Admin)

```bash
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin-token>" \
  -d '{
    "sku": "TSHIRT-BLU-M",
    "name": "Blue T-Shirt (Medium)",
    "description": "Comfortable cotton t-shirt in blue",
    "category_id": "<category-uuid>",
    "price": 29.99,
    "compare_at_price": 39.99,
    "quantity": 100,
    "status": "active",
    "images": ["https://example.com/blue-tshirt.jpg"]
  }'
```

### Add to Cart

```bash
curl -X POST http://localhost:8080/cart/items \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": "<product-uuid>",
    "quantity": 2
  }'
```

### Checkout (Create Order)

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "shipping_address_id": "<address-uuid>",
    "billing_address_id": "<address-uuid>",
    "shipping_method": "standard",
    "payment_method_id": "pm_card_visa",
    "notes": "Please leave at door"
  }'
```

### Track Order

```bash
curl http://localhost:8080/orders/ORD-2026-0001 \
  -H "Authorization: Bearer <token>"
```

**Expected Response:**
```json
{
  "order_number": "ORD-2026-0001",
  "status": "shipped",
  "payment_status": "captured",
  "total": 64.98,
  "tracking_number": "1Z999AA10123456784",
  "shipped_at": "2026-01-12T14:30:00Z",
  "items": [
    {
      "product_name": "Blue T-Shirt (Medium)",
      "quantity": 2,
      "unit_price": 29.99,
      "total_price": 59.98
    }
  ]
}
```

### Request Refund

```bash
curl -X POST http://localhost:8080/orders/ORD-2026-0001/refund \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "amount": 29.99,
    "reason": "Item did not fit",
    "return_items": true
  }'
```

## Events

### Order Events

| Event | Trigger | Published To |
|-------|---------|--------------|
| `OrderPlaced` | New order created | Kafka: orders |
| `OrderConfirmed` | Payment successful | Webhook, Kafka |
| `OrderShipped` | Shipment created | Webhook, Kafka |
| `OrderCancelled` | Order cancelled | - |
| `OrderFailed` | Processing failed | Slack: #order-alerts |

### Refund Events

| Event | Trigger | Published To |
|-------|---------|--------------|
| `RefundRequested` | Refund initiated | - |
| `RefundProcessed` | Refund successful | Webhook |
| `RefundFailed` | Refund failed | Slack: #order-alerts |

### Inventory Events

| Event | Trigger | Published To |
|-------|---------|--------------|
| `LowStockAlert` | Stock below threshold | Slack, Webhook |

## Key Concepts Demonstrated

1. **Saga Pattern**: Multi-step workflow with automatic compensation
2. **Circuit Breaker**: Prevents cascading failures to payment gateway
3. **Exponential Backoff**: Smart retry strategy for transient failures
4. **Event-Driven Architecture**: Loose coupling via events
5. **Inventory Reservations**: Prevents overselling
6. **Payment Integration**: Full payment lifecycle
7. **Shipping Integration**: Rate calculation and label generation
8. **Admin Operations**: Status management and reporting

## Next Steps

- See [Integrations](../04-integrations/) for more integration patterns
- Try [Scheduled Jobs](../05-scheduled-jobs/) for background tasks
