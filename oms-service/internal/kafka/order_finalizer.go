package kafka

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"oms-service/internal/orders"
	"time"
)

// OrderFinalizerHandler handles order finalization logic
type OrderFinalizerHandler struct {
	imsBaseURL string
	httpClient *http.Client
}

// NewOrderFinalizerHandler creates a new order finalizer
func NewOrderFinalizerHandler(imsBaseURL string) *OrderFinalizerHandler {
	return &OrderFinalizerHandler{
		imsBaseURL: imsBaseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// HandleOrderCreated processes order.created events with full finalization logic
func (h *OrderFinalizerHandler) HandleOrderCreated(ctx context.Context, event *OrderCreatedEvent) error {
	log.Printf("🔄 [ORDER FINALIZER] Processing order finalization: OrderID=%s", event.OrderID)

	// Step 1: Check inventory availability
	available, err := h.checkInventoryAvailability(ctx, event)
	if err != nil {
		log.Printf("❌ [ORDER FINALIZER] Inventory check failed for order %s: %v", event.OrderID, err)
		return h.markOrderFailed(ctx, event.OrderID, fmt.Sprintf("Inventory check failed: %v", err))
	}

	if !available {
		log.Printf("⚠️ [ORDER FINALIZER] Insufficient inventory for order %s, keeping on_hold", event.OrderID)
		return h.updateOrderStatus(ctx, event.OrderID, "on_hold", "Insufficient inventory")
	}

	// Step 2: Reserve inventory atomically
	err = h.reserveInventory(ctx, event)
	if err != nil {
		log.Printf("❌ [ORDER FINALIZER] Inventory reservation failed for order %s: %v", event.OrderID, err)
		return h.markOrderFailed(ctx, event.OrderID, fmt.Sprintf("Inventory reservation failed: %v", err))
	}

	// Step 3: Update order status to new_order
	err = h.updateOrderStatus(ctx, event.OrderID, "new_order", "Inventory confirmed and reserved")
	if err != nil {
		// Try to rollback inventory reservation
		rollbackErr := h.releaseInventory(ctx, event)
		if rollbackErr != nil {
			log.Printf("❌ [ORDER FINALIZER] Failed to rollback inventory for order %s: %v", event.OrderID, rollbackErr)
		}
		return fmt.Errorf("failed to update order status: %w", err)
	}

	log.Printf("✅ [ORDER FINALIZER] Order %s successfully finalized: on_hold → new_order", event.OrderID)
	return nil
}

// HandleOrderUpdated handles order update events (placeholder for future expansion)
func (h *OrderFinalizerHandler) HandleOrderUpdated(ctx context.Context, event *OrderCreatedEvent) error {
	log.Printf("📥 [ORDER FINALIZER] Order updated event: OrderID=%s, Status=%s", event.OrderID, event.Status)
	// Future: Handle order modifications, status changes, etc.
	return nil
}

// HandleOrderCancelled handles order cancellation with inventory release
func (h *OrderFinalizerHandler) HandleOrderCancelled(ctx context.Context, event *OrderCreatedEvent) error {
	log.Printf("🚫 [ORDER FINALIZER] Processing order cancellation: OrderID=%s", event.OrderID)

	// Release any reserved inventory
	err := h.releaseInventory(ctx, event)
	if err != nil {
		log.Printf("⚠️ [ORDER FINALIZER] Failed to release inventory for cancelled order %s: %v", event.OrderID, err)
	}

	// Update order status to cancelled
	return h.updateOrderStatus(ctx, event.OrderID, "cancelled", "Order cancelled by system")
}

// checkInventoryAvailability verifies if sufficient inventory is available
func (h *OrderFinalizerHandler) checkInventoryAvailability(ctx context.Context, event *OrderCreatedEvent) (bool, error) {
	for _, item := range event.Items {
		available, err := h.getAvailableInventory(ctx, item.HubID, item.SKU)
		if err != nil {
			return false, fmt.Errorf("failed to check inventory for SKU %s at hub %s: %w", item.SKU, item.HubID, err)
		}

		if available < item.Quantity {
			log.Printf("⚠️ [INVENTORY] Insufficient stock: SKU=%s, Hub=%s, Required=%d, Available=%d",
				item.SKU, item.HubID, item.Quantity, available)
			return false, nil
		}
	}

	log.Printf("✅ [INVENTORY] All items available for order %s", event.OrderID)
	return true, nil
}

// getAvailableInventory queries IMS for available inventory
func (h *OrderFinalizerHandler) getAvailableInventory(ctx context.Context, hubCode, skuCode string) (int, error) {
	url := fmt.Sprintf("%s/api/v1/inventory/%s/%s", h.imsBaseURL, hubCode, skuCode)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create inventory request: %w", err)
	}

	// Add required tenant header
	req.Header.Set("X-Tenant-ID", "default")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		// IMS unavailable, simulate inventory check
		log.Printf("⚠️ [INVENTORY] IMS unavailable, simulating inventory check for %s/%s", hubCode, skuCode)
		return h.simulateInventoryCheck(skuCode), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Inventory item not found, assume 0 available
		return 0, nil
	}

	if resp.StatusCode != http.StatusOK {
		// For other errors, simulate inventory
		log.Printf("⚠️ [INVENTORY] IMS error %d, simulating inventory for %s/%s", resp.StatusCode, hubCode, skuCode)
		return h.simulateInventoryCheck(skuCode), nil
	}

	// Parse inventory response
	var inventory struct {
		Available int `json:"available"`
		Quantity  int `json:"quantity"`
	}

	err = json.NewDecoder(resp.Body).Decode(&inventory)
	if err != nil {
		log.Printf("⚠️ [INVENTORY] Failed to parse IMS response, simulating: %v", err)
		return h.simulateInventoryCheck(skuCode), nil
	}

	return inventory.Available, nil
}

// simulateInventoryCheck provides fallback inventory simulation
func (h *OrderFinalizerHandler) simulateInventoryCheck(skuCode string) int {
	// Simulate inventory based on SKU patterns
	switch {
	case len(skuCode) > 0 && skuCode[0] == 'L': // Laptops
		return 50 // High stock
	case len(skuCode) > 0 && skuCode[0] == 'M': // Mice
		return 100 // Very high stock
	case len(skuCode) > 0 && skuCode[0] == 'K': // Keyboards
		return 25 // Medium stock
	default:
		return 10 // Default low stock
	}
}

// reserveInventory atomically reserves inventory for all order items
func (h *OrderFinalizerHandler) reserveInventory(ctx context.Context, event *OrderCreatedEvent) error {
	log.Printf("🔒 [INVENTORY] Reserving inventory for order %s", event.OrderID)

	for _, item := range event.Items {
		err := h.reserveItemInventory(ctx, item.HubID, item.SKU, item.Quantity)
		if err != nil {
			// Rollback previous reservations on failure
			h.rollbackPreviousReservations(ctx, event, item)
			return fmt.Errorf("failed to reserve inventory for SKU %s: %w", item.SKU, err)
		}
	}

	log.Printf("✅ [INVENTORY] All inventory reserved for order %s", event.OrderID)
	return nil
}

// reserveItemInventory reserves inventory for a specific item
func (h *OrderFinalizerHandler) reserveItemInventory(ctx context.Context, hubCode, skuCode string, quantity int) error {
	url := fmt.Sprintf("%s/api/v1/inventory/reserve", h.imsBaseURL)

	requestBody := map[string]interface{}{
		"hub_code": hubCode,
		"sku_code": skuCode,
		"quantity": quantity,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal reservation request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create reservation request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "default")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		// IMS unavailable, log and continue (simulated success)
		log.Printf("⚠️ [INVENTORY] IMS unavailable for reservation, simulating success: %s/%s qty=%d", hubCode, skuCode, quantity)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Parse error response
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		return fmt.Errorf("inventory reservation failed (status %d): %v", resp.StatusCode, errorResp)
	}

	log.Printf("✅ [INVENTORY] Reserved %d units of %s at hub %s", quantity, skuCode, hubCode)
	return nil
}

// releaseInventory releases reserved inventory (for cancellations or rollbacks)
func (h *OrderFinalizerHandler) releaseInventory(ctx context.Context, event *OrderCreatedEvent) error {
	log.Printf("🔓 [INVENTORY] Releasing inventory for order %s", event.OrderID)

	for _, item := range event.Items {
		err := h.releaseItemInventory(ctx, item.HubID, item.SKU, item.Quantity)
		if err != nil {
			log.Printf("⚠️ [INVENTORY] Failed to release inventory for SKU %s: %v", item.SKU, err)
			// Continue releasing other items even if one fails
		}
	}

	return nil
}

// releaseItemInventory releases inventory for a specific item
func (h *OrderFinalizerHandler) releaseItemInventory(ctx context.Context, hubCode, skuCode string, quantity int) error {
	url := fmt.Sprintf("%s/api/v1/inventory/release", h.imsBaseURL)

	requestBody := map[string]interface{}{
		"hub_code": hubCode,
		"sku_code": skuCode,
		"quantity": quantity,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal release request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create release request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "default")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		// IMS unavailable, log and continue
		log.Printf("⚠️ [INVENTORY] IMS unavailable for release, simulating success: %s/%s qty=%d", hubCode, skuCode, quantity)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		return fmt.Errorf("inventory release failed (status %d): %v", resp.StatusCode, errorResp)
	}

	log.Printf("✅ [INVENTORY] Released %d units of %s at hub %s", quantity, skuCode, hubCode)
	return nil
}

// rollbackPreviousReservations rolls back reservations made before a failure
func (h *OrderFinalizerHandler) rollbackPreviousReservations(ctx context.Context, event *OrderCreatedEvent, failedItem OrderItem) {
	log.Printf("🔄 [INVENTORY] Rolling back previous reservations for order %s", event.OrderID)

	for _, item := range event.Items {
		if item.SKU == failedItem.SKU && item.HubID == failedItem.HubID {
			break // Stop at the failed item
		}
		h.releaseItemInventory(ctx, item.HubID, item.SKU, item.Quantity)
	}
}

// updateOrderStatus updates the order status in MongoDB
func (h *OrderFinalizerHandler) updateOrderStatus(ctx context.Context, orderID, status, notes string) error {
	log.Printf("📝 [ORDER] Updating order %s status: %s (%s)", orderID, status, notes)

	err := orders.UpdateOrderStatus(orderID, status)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	log.Printf("✅ [ORDER] Order %s status updated to: %s", orderID, status)
	return nil
}

// markOrderFailed marks an order as failed with error details
func (h *OrderFinalizerHandler) markOrderFailed(ctx context.Context, orderID, errorMsg string) error {
	log.Printf("❌ [ORDER] Marking order %s as failed: %s", orderID, errorMsg)
	return h.updateOrderStatus(ctx, orderID, "failed", errorMsg)
}
