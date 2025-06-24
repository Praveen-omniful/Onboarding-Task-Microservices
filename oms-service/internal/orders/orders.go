package orders

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Order represents an order in the system
type Order struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrderID         string             `bson:"order_id" json:"order_id"`
	CustomerID      string             `bson:"customer_id" json:"customer_id"`
	CustomerName    string             `bson:"customer_name" json:"customer_name"`
	CustomerEmail   string             `bson:"customer_email" json:"customer_email"`
	ProductName     string             `bson:"product_name" json:"product_name"`
	ProductSKU      string             `bson:"product_sku" json:"product_sku"`
	Quantity        int                `bson:"quantity" json:"quantity"`
	UnitPrice       float64            `bson:"unit_price" json:"unit_price"`
	TotalAmount     float64            `bson:"total_amount" json:"total_amount"`
	HubID           string             `bson:"hub_id" json:"hub_id"`
	ShippingAddress string             `bson:"shipping_address" json:"shipping_address"`
	OrderDate       string             `bson:"order_date" json:"order_date"`
	Status          string             `bson:"status" json:"status"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at" json:"updated_at"`
}

// OrderStats represents order statistics
type OrderStats struct {
	TotalOrders   int64   `json:"total_orders"`
	TotalRevenue  float64 `json:"total_revenue"`
	AverageOrder  float64 `json:"average_order"`
	LastProcessed string  `json:"last_processed"`
}

var (
	mongoClient      *mongo.Client
	ordersCollection *mongo.Collection
)

// InitializeMongoDB initializes the MongoDB connection and collection
func InitializeMongoDB(connectionURI string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectionURI))
	if err != nil {
		return err
	}

	// Test the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return err
	}

	mongoClient = client
	ordersCollection = client.Database("oms_database").Collection("orders")

	log.Println("✅ MongoDB connected successfully")
	log.Println("📊 Database: oms_database, Collection: orders")

	return nil
}

// CreateOrder creates a new order in MongoDB
func CreateOrder(order *Order) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Set timestamps
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()

	// Calculate total amount if not set
	if order.TotalAmount == 0 {
		order.TotalAmount = order.UnitPrice * float64(order.Quantity)
	}

	// Set default status if not set
	if order.Status == "" {
		order.Status = "pending"
	}

	// Insert the order
	result, err := ordersCollection.InsertOne(ctx, order)
	if err != nil {
		log.Printf("❌ Failed to create order: %v", err)
		return err
	}

	order.ID = result.InsertedID.(primitive.ObjectID)
	log.Printf("✅ Order created successfully: %s", order.OrderID)

	return nil
}

// GetAllOrders retrieves all orders from MongoDB
func GetAllOrders() ([]Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := ordersCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var orders []Order
	err = cursor.All(ctx, &orders)
	if err != nil {
		return nil, err
	}

	return orders, nil
}

// GetOrderByID retrieves an order by its ID
func GetOrderByID(orderID string) (*Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var order Order
	err := ordersCollection.FindOne(ctx, bson.M{"order_id": orderID}).Decode(&order)
	if err != nil {
		return nil, err
	}

	return &order, nil
}

// UpdateOrderStatus updates the status of an order
func UpdateOrderStatus(orderID string, status string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	_, err := ordersCollection.UpdateOne(ctx, bson.M{"order_id": orderID}, update)
	if err != nil {
		log.Printf("❌ Failed to update order status: %v", err)
		return err
	}

	log.Printf("✅ Order %s status updated to: %s", orderID, status)
	return nil
}

// GetOrderStats retrieves order statistics
func GetOrderStats() (*OrderStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Count total orders
	totalOrders, err := ordersCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	// Calculate total revenue and average order
	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id":           nil,
				"total_revenue": bson.M{"$sum": "$total_amount"},
				"avg_order":     bson.M{"$avg": "$total_amount"},
			},
		},
	}

	cursor, err := ordersCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result []bson.M
	err = cursor.All(ctx, &result)
	if err != nil {
		return nil, err
	}

	stats := &OrderStats{
		TotalOrders:   totalOrders,
		LastProcessed: time.Now().Format(time.RFC3339),
	}

	if len(result) > 0 {
		if totalRevenue, ok := result[0]["total_revenue"].(float64); ok {
			stats.TotalRevenue = totalRevenue
		}
		if avgOrder, ok := result[0]["avg_order"].(float64); ok {
			stats.AverageOrder = avgOrder
		}
	}

	return stats, nil
}

// CreateSampleOrders creates sample orders for testing
func CreateSampleOrders() error {
	sampleOrders := []Order{
		{
			OrderID:         "ORD001",
			CustomerID:      "CUST001",
			CustomerName:    "John Doe",
			CustomerEmail:   "john@example.com",
			ProductName:     "Gaming Laptop",
			ProductSKU:      "LAP-GAME-001",
			Quantity:        1,
			UnitPrice:       1299.99,
			TotalAmount:     1299.99,
			HubID:           "HUB001",
			ShippingAddress: "123 Main St, New York, NY 10001",
			OrderDate:       "2024-01-15",
			Status:          "pending",
		},
		{
			OrderID:         "ORD002",
			CustomerID:      "CUST002",
			CustomerName:    "Jane Smith",
			CustomerEmail:   "jane@example.com",
			ProductName:     "Wireless Mouse",
			ProductSKU:      "MOU-WIR-002",
			Quantity:        2,
			UnitPrice:       39.99,
			TotalAmount:     79.98,
			HubID:           "HUB001",
			ShippingAddress: "456 Oak Ave, Los Angeles, CA 90210",
			OrderDate:       "2024-01-16",
			Status:          "processing",
		},
		{
			OrderID:         "ORD003",
			CustomerID:      "CUST003",
			CustomerName:    "Bob Wilson",
			CustomerEmail:   "bob@example.com",
			ProductName:     "Mechanical Keyboard",
			ProductSKU:      "KEY-MECH-003",
			Quantity:        1,
			UnitPrice:       129.99,
			TotalAmount:     129.99,
			HubID:           "HUB002",
			ShippingAddress: "789 Pine Rd, Chicago, IL 60601",
			OrderDate:       "2024-01-17",
			Status:          "shipped",
		},
	}

	for _, order := range sampleOrders {
		err := CreateOrder(&order)
		if err != nil {
			log.Printf("❌ Failed to create sample order %s: %v", order.OrderID, err)
			return err
		}
	}

	log.Printf("✅ Created %d sample orders", len(sampleOrders))
	return nil
}

// DeleteAllOrders deletes all orders (for testing purposes)
func DeleteAllOrders() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := ordersCollection.DeleteMany(ctx, bson.M{})
	if err != nil {
		return err
	}

	log.Printf("🗑️ Deleted %d orders", result.DeletedCount)
	return nil
}

// GetMongoClient returns the MongoDB client
func GetMongoClient() *mongo.Client {
	return mongoClient
}
