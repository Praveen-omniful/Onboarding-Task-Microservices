#!/bin/bash

echo "🎯 FINAL TEST - OMS Microservices Flow"
echo "======================================"

SERVICE_URL="http://localhost:8088"

# Get initial stats
echo "📊 Getting initial stats..."
RESPONSE=$(curl -s "$SERVICE_URL/stats" || echo "{\"total_orders\":0}")
echo "Response: $RESPONSE"

INITIAL_COUNT=$(echo "$RESPONSE" | grep -o '"total_orders":[0-9]*' | cut -d':' -f2)
if [ -z "$INITIAL_COUNT" ]; then
    INITIAL_COUNT=0
fi

echo "Initial order count: $INITIAL_COUNT"

# Upload CSV
echo ""
echo "📤 Uploading test CSV..."
curl -X POST -F "file=@test_proper_format.csv" "$SERVICE_URL/upload"

echo ""
echo ""
echo "⏳ Waiting 20 seconds for processing..."
sleep 20

# Get final stats
echo "📊 Getting final stats..."
FINAL_RESPONSE=$(curl -s "$SERVICE_URL/stats" || echo "{\"total_orders\":0}")
echo "Response: $FINAL_RESPONSE"

FINAL_COUNT=$(echo "$FINAL_RESPONSE" | grep -o '"total_orders":[0-9]*' | cut -d':' -f2)
if [ -z "$FINAL_COUNT" ]; then
    FINAL_COUNT=0
fi

echo "Final order count: $FINAL_COUNT"

# Calculate difference
DIFFERENCE=$((FINAL_COUNT - INITIAL_COUNT))
echo ""
echo "📈 Orders added: $DIFFERENCE"

if [ $DIFFERENCE -gt 0 ]; then
    echo "✅ SUCCESS! The microservices flow is working!"
    echo "🎉 CSV → S3 → SQS → Processing → MongoDB → Stats update"
else
    echo "❌ No orders were added. Check logs for issues."
fi

echo ""
echo "🔍 For more details:"
echo "   curl $SERVICE_URL/orders | head -20"
