#!/bin/bash
# =============================================================================
# E-commerce Backend Test Script
# =============================================================================
# Run this script to test the E-commerce API endpoints
# Usage: ./test.sh [BASE_URL] [ADMIN_TOKEN]
# =============================================================================

BASE_URL="${1:-http://localhost:8080}"
ADMIN_TOKEN="${2:-admin-token}"
CUSTOMER_TOKEN=""

echo "============================================="
echo "E-commerce Backend Test Script"
echo "Base URL: $BASE_URL"
echo "============================================="
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

test_header() {
    echo -e "${BLUE}>>> $1${NC}"
}

section_header() {
    echo ""
    echo -e "${YELLOW}=============================================${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}=============================================${NC}"
    echo ""
}

# Store IDs
CUSTOMER_ID=""
ADDRESS_ID=""
CATEGORY_ID=""
PRODUCT_ID=""
CART_ID=""
ORDER_NUMBER=""

# =============================================================================
section_header "1. Customer Registration"
# =============================================================================

test_header "Register Customer"
CUSTOMER_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "customer@example.com",
    "password": "password123",
    "first_name": "John",
    "last_name": "Doe",
    "phone": "+1-555-0100"
  }')
echo "$CUSTOMER_RESPONSE" | jq .
CUSTOMER_ID=$(echo "$CUSTOMER_RESPONSE" | jq -r '.id')
echo "Customer ID: $CUSTOMER_ID"
echo ""

test_header "Login Customer"
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "customer@example.com",
    "password": "password123"
  }')
CUSTOMER_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token')
echo "Customer Token: ${CUSTOMER_TOKEN:0:50}..."
echo ""

# =============================================================================
section_header "2. Add Customer Address"
# =============================================================================

test_header "Add Shipping Address"
ADDRESS_RESPONSE=$(curl -s -X POST "$BASE_URL/addresses" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -d '{
    "type": "both",
    "is_default": true,
    "label": "Home",
    "street_line1": "123 Main Street",
    "street_line2": "Apt 4B",
    "city": "San Francisco",
    "state": "CA",
    "postal_code": "94105",
    "country": "US",
    "recipient_name": "John Doe",
    "recipient_phone": "+1-555-0100"
  }')
echo "$ADDRESS_RESPONSE" | jq .
ADDRESS_ID=$(echo "$ADDRESS_RESPONSE" | jq -r '.id')
echo "Address ID: $ADDRESS_ID"
echo ""

# =============================================================================
section_header "3. Create Product Catalog (Admin)"
# =============================================================================

test_header "Create Category: Clothing"
CATEGORY_RESPONSE=$(curl -s -X POST "$BASE_URL/categories" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "name": "Clothing",
    "slug": "clothing",
    "description": "Apparel and clothing items"
  }')
echo "$CATEGORY_RESPONSE" | jq .
CATEGORY_ID=$(echo "$CATEGORY_RESPONSE" | jq -r '.id')
echo ""

test_header "Create Product: Blue T-Shirt"
PRODUCT_RESPONSE=$(curl -s -X POST "$BASE_URL/products" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d "{
    \"sku\": \"TSHIRT-BLU-M\",
    \"name\": \"Blue T-Shirt (Medium)\",
    \"description\": \"Comfortable cotton t-shirt in a beautiful shade of blue. Perfect for everyday wear.\",
    \"short_description\": \"Cotton t-shirt in blue\",
    \"category_id\": \"$CATEGORY_ID\",
    \"brand\": \"CodeAI Apparel\",
    \"tags\": [\"cotton\", \"blue\", \"casual\", \"medium\"],
    \"price\": 29.99,
    \"compare_at_price\": 39.99,
    \"cost_price\": 12.00,
    \"track_inventory\": true,
    \"quantity\": 100,
    \"status\": \"active\",
    \"is_featured\": true,
    \"images\": [\"https://example.com/blue-tshirt-1.jpg\", \"https://example.com/blue-tshirt-2.jpg\"]
  }")
echo "$PRODUCT_RESPONSE" | jq .
PRODUCT_ID=$(echo "$PRODUCT_RESPONSE" | jq -r '.id')
PRODUCT_SKU=$(echo "$PRODUCT_RESPONSE" | jq -r '.sku')
echo "Product ID: $PRODUCT_ID"
echo ""

test_header "Create Product: Red T-Shirt"
curl -s -X POST "$BASE_URL/products" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d "{
    \"sku\": \"TSHIRT-RED-M\",
    \"name\": \"Red T-Shirt (Medium)\",
    \"description\": \"Comfortable cotton t-shirt in vibrant red.\",
    \"category_id\": \"$CATEGORY_ID\",
    \"price\": 29.99,
    \"quantity\": 50,
    \"status\": \"active\"
  }" | jq '.sku, .name, .price'
echo ""

# =============================================================================
section_header "4. Browse Products"
# =============================================================================

test_header "List All Products"
curl -s "$BASE_URL/products" | jq '.data | length, .[0].name'
echo ""

test_header "Search Products: 'blue'"
curl -s "$BASE_URL/products?search=blue" | jq '.data | .[0].name'
echo ""

test_header "Filter by Category: clothing"
curl -s "$BASE_URL/products?category=clothing" | jq '.data | length'
echo ""

test_header "Get Product by SKU"
curl -s "$BASE_URL/products/$PRODUCT_SKU" | jq '.name, .price, .quantity'
echo ""

# =============================================================================
section_header "5. Shopping Cart Operations"
# =============================================================================

test_header "Get Current Cart (empty)"
curl -s "$BASE_URL/cart" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq '.total'
echo ""

test_header "Add Blue T-Shirt to Cart (qty: 2)"
CART_RESPONSE=$(curl -s -X POST "$BASE_URL/cart/items" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -d "{
    \"product_id\": \"$PRODUCT_ID\",
    \"quantity\": 2
  }")
echo "$CART_RESPONSE" | jq '.subtotal, .total, .items | length'
CART_ID=$(echo "$CART_RESPONSE" | jq -r '.id')
CART_ITEM_ID=$(echo "$CART_RESPONSE" | jq -r '.items[0].id')
echo ""

test_header "Update Cart Item Quantity to 3"
curl -s -X PUT "$BASE_URL/cart/items/$CART_ITEM_ID" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -d '{"quantity": 3}' | jq '.subtotal, .total'
echo ""

test_header "Apply Coupon: SAVE10"
curl -s -X POST "$BASE_URL/cart/coupon" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -d '{"code": "SAVE10"}' | jq '.coupon_code, .discount_total, .total'
echo ""

test_header "View Final Cart"
curl -s "$BASE_URL/cart" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq .
echo ""

# =============================================================================
section_header "6. Checkout (Create Order)"
# =============================================================================

test_header "Create Order from Cart"
ORDER_RESPONSE=$(curl -s -X POST "$BASE_URL/orders" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -d "{
    \"shipping_address_id\": \"$ADDRESS_ID\",
    \"billing_address_id\": \"$ADDRESS_ID\",
    \"shipping_method\": \"standard\",
    \"payment_method_id\": \"pm_card_visa\",
    \"notes\": \"Please leave at the front door\"
  }")
echo "$ORDER_RESPONSE" | jq .
ORDER_NUMBER=$(echo "$ORDER_RESPONSE" | jq -r '.order_number')
ORDER_ID=$(echo "$ORDER_RESPONSE" | jq -r '.id')
echo "Order Number: $ORDER_NUMBER"
echo ""

# =============================================================================
section_header "7. Order Tracking"
# =============================================================================

test_header "Get Order Details"
curl -s "$BASE_URL/orders/$ORDER_NUMBER" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq '.order_number, .status, .payment_status, .total'
echo ""

test_header "List Customer Orders"
curl -s "$BASE_URL/orders" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq '.data | length, .[0].order_number, .[0].status'
echo ""

# Wait for order processing (simulated)
echo "Waiting for order processing (simulated)..."
sleep 2
echo ""

test_header "Check Order Status After Processing"
curl -s "$BASE_URL/orders/$ORDER_NUMBER" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq '.status, .payment_status, .tracking_number'
echo ""

# =============================================================================
section_header "8. Admin Operations"
# =============================================================================

test_header "List All Orders (Admin)"
curl -s "$BASE_URL/admin/orders" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.data | length, .[0].order_number, .[0].total'
echo ""

test_header "Filter Orders by Status: confirmed"
curl -s "$BASE_URL/admin/orders?status=confirmed" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.data | length'
echo ""

test_header "Update Order Status to 'processing'"
curl -s -X PUT "$BASE_URL/admin/orders/$ORDER_ID/status" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "status": "processing",
    "internal_notes": "Order picked and ready for packing"
  }' | jq '.status'
echo ""

# =============================================================================
section_header "9. Refund Request"
# =============================================================================

test_header "Request Partial Refund"
REFUND_RESPONSE=$(curl -s -X POST "$BASE_URL/orders/$ORDER_NUMBER/refund" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -d '{
    "amount": 29.99,
    "reason": "One item did not fit properly",
    "return_items": true
  }')
echo "$REFUND_RESPONSE" | jq '.payment_status'
echo ""

test_header "Check Order After Refund"
curl -s "$BASE_URL/orders/$ORDER_NUMBER" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq '.payment_status, .refunded_at'
echo ""

# =============================================================================
section_header "10. Inventory Check"
# =============================================================================

test_header "Check Product Inventory After Order"
curl -s "$BASE_URL/products/$PRODUCT_SKU" | jq '.quantity'
echo "Expected: Reduced by order quantity (minus any returns)"
echo ""

# =============================================================================
section_header "11. Cancel Order Test"
# =============================================================================

# Create another order to cancel
test_header "Create Another Order"
curl -s -X POST "$BASE_URL/cart/items" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -d "{\"product_id\": \"$PRODUCT_ID\", \"quantity\": 1}" > /dev/null

ORDER2_RESPONSE=$(curl -s -X POST "$BASE_URL/orders" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -d "{
    \"shipping_address_id\": \"$ADDRESS_ID\",
    \"billing_address_id\": \"$ADDRESS_ID\",
    \"shipping_method\": \"express\",
    \"payment_method_id\": \"pm_card_visa\"
  }")
ORDER2_NUMBER=$(echo "$ORDER2_RESPONSE" | jq -r '.order_number')
echo "Order 2 Number: $ORDER2_NUMBER"
echo ""

test_header "Cancel Order"
curl -s -X POST "$BASE_URL/orders/$ORDER2_NUMBER/cancel" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -d '{
    "reason": "Changed my mind"
  }' | jq '.status, .cancelled_at'
echo ""

# =============================================================================
section_header "Test Complete!"
# =============================================================================

echo "Summary:"
echo "  - Customer created: customer@example.com"
echo "  - Address created: $ADDRESS_ID"
echo "  - Products created: 2 (Blue T-Shirt, Red T-Shirt)"
echo "  - Orders:"
echo "    - $ORDER_NUMBER (completed with partial refund)"
echo "    - $ORDER2_NUMBER (cancelled)"
echo ""
echo "Workflow Events Triggered:"
echo "  - OrderPlaced → OrderProcessing workflow"
echo "  - OrderConfirmed (payment successful)"
echo "  - OrderReadyForFulfillment → OrderFulfillment workflow"
echo "  - RefundRequested → ProcessRefund workflow"
echo "  - OrderCancelled"
echo ""
echo "To check events:"
echo "  - Check Kafka topic: orders"
echo "  - Check Slack channel: #order-alerts"
echo "  - Check configured webhooks"
