package mongodb

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB client configuration
type Config struct {
	URI        string
	Database   string
	Collection string
}

// MongoDB client wrapper using standard MongoDB driver
type Client struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
	config     Config
}

// NewClient creates a new MongoDB client using standard MongoDB driver
func NewClient(config Config) (*Client, error) {
	log.Printf("🔌 Connecting to MongoDB: %s/%s", config.URI, config.Database)

	// Set client options
	clientOptions := options.Client().ApplyURI(config.URI)

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Printf("⚠️  MongoDB ping failed (MongoDB not available?): %v", err)
		// Don't fail, just warn - might be running without MongoDB for demo
	} else {
		log.Printf("✅ MongoDB connection successful")
	}

	// Get database and collection
	db := client.Database(config.Database)
	collection := db.Collection(config.Collection)

	mongoClient := &Client{
		client:     client,
		db:         db,
		collection: collection,
		config:     config,
	}

	return mongoClient, nil
}

// InsertOrder inserts an order document into MongoDB
func (c *Client) InsertOrder(ctx context.Context, order interface{}) error {
	result, err := c.collection.InsertOne(ctx, order)
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	log.Printf("✅ Order inserted to MongoDB with ID: %v", result.InsertedID)
	return nil
}

// FindOrderByID finds an order by its ID
func (c *Client) FindOrderByID(ctx context.Context, orderID string) (bson.M, error) {
	var result bson.M
	err := c.collection.FindOne(ctx, bson.M{"order_id": orderID}).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to find order %s: %w", orderID, err)
	}

	return result, nil
}

// UpdateOrderStatus updates the status of an order
func (c *Client) UpdateOrderStatus(ctx context.Context, orderID, status string) error {
	filter := bson.M{"order_id": orderID}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	result, err := c.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update order %s: %w", orderID, err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("order %s not found", orderID)
	}

	log.Printf("✅ Order %s status updated to: %s", orderID, status)
	return nil
}

// GetOrderCount returns the total number of orders in the collection
func (c *Client) GetOrderCount(ctx context.Context) (int64, error) {
	count, err := c.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, fmt.Errorf("failed to count orders: %w", err)
	}

	return count, nil
}

// FindOrdersByStatus finds orders by their status
func (c *Client) FindOrdersByStatus(ctx context.Context, status string) ([]bson.M, error) {
	filter := bson.M{"status": status}
	cursor, err := c.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find orders with status %s: %w", status, err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode orders: %w", err)
	}

	return results, nil
}

// InsertManyOrders inserts multiple orders in a batch
func (c *Client) InsertManyOrders(ctx context.Context, orders []interface{}) error {
	if len(orders) == 0 {
		return nil
	}

	result, err := c.collection.InsertMany(ctx, orders)
	if err != nil {
		return fmt.Errorf("failed to insert batch orders: %w", err)
	}

	log.Printf("✅ Inserted %d orders to MongoDB", len(result.InsertedIDs))
	return nil
}

// CreateIndexes creates necessary indexes for the orders collection
func (c *Client) CreateIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{"order_id", 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{"status", 1}},
		},
		{
			Keys: bson.D{{"customer_email", 1}},
		},
		{
			Keys: bson.D{{"created_at", 1}},
		},
	}

	_, err := c.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	log.Printf("✅ MongoDB indexes created successfully")
	return nil
}

// GetCollection returns the MongoDB collection for direct access
func (c *Client) GetCollection() *mongo.Collection {
	return c.collection
}

// Close closes the MongoDB connection
func (c *Client) Close(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}
