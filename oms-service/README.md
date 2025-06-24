# OMS (Order Management Service)

A production-ready microservice for processing large CSV order files with real-time validation, MongoDB storage, and Kafka event streaming.

## ✨ Features

- **📁 CSV Processing**: Batch processing of large CSV files using go_commons
- **☁️ Cloud Storage**: S3 integration via LocalStack for file storage
- **📬 Message Queue**: SQS integration for asynchronous processing
- **💾 Database**: MongoDB for order storage with indexing and aggregation
- **✅ Validation**: Real-time SKU/Hub validation via IMS service APIs
- **📤 Event Streaming**: Kafka event emission for order lifecycle management
- **🚨 Error Handling**: Invalid record logging with downloadable CSV files
- **🌐 REST API**: HTTP endpoints for upload, statistics, and file management
- **🐳 Containerized**: Docker Compose setup for all dependencies

## 🚀 Quick Start

### Option 1: Automated Setup (Recommended)

```bash
# Run the setup script
./setup.sh    # Linux/Mac
setup.bat     # Windows

# Start the OMS service
go run cmd/server/main.go

# Test with sample data
./test.sh     # Linux/Mac
```

### Option 2: Manual Setup

1. **Start Dependencies**
   ```bash
   docker-compose up -d
   ```

2. **Install Dependencies**
   ```bash
   go mod tidy
   ```

3. **Start OMS Service**
   ```bash
   go run cmd/server/main.go
   ```

4. **Upload Test File**
   ```bash
   curl -X POST -F "file=@sample_orders.csv" http://localhost:8080/upload
   ```

## 📖 Complete Guide

For detailed setup instructions, CSV format requirements, API documentation, and troubleshooting, see **[GETTING_STARTED.md](GETTING_STARTED.md)**.

## 🏗️ Architecture

```
CSV Upload → S3 Storage → SQS Message → Processing Pipeline → MongoDB + Kafka Events
     ↓              ↓            ↓              ↓                ↓
HTTP API      LocalStack    go_commons    IMS Validation    Event Streaming
```

## 📊 API Endpoints

- `POST /upload` - Upload CSV order file
- `GET /stats` - View order statistics and counts
- `GET /invalid-files` - List invalid record files
- `GET /invalid-files/{name}` - Download invalid records

## 🧪 Testing

```bash
# Upload sample orders
curl -X POST -F "file=@sample_orders.csv" http://localhost:8080/upload

# Check processing results
curl http://localhost:8080/stats

# View invalid records (if any)
curl http://localhost:8080/invalid-files
```

## 🔧 Configuration

The service auto-configures for local development:
- **MongoDB**: `localhost:27018`
- **Kafka**: `localhost:9092`
- **LocalStack**: `localhost:4566`
- **OMS API**: `localhost:8080`
- **IMS Service**: `localhost:8081` (for validation)

## 📝 CSV Format

Required columns (order doesn't matter):
```csv
order_id,customer_name,customer_email,product_name,sku,hub_id,quantity,unit_price,total_amount,order_date,shipping_address
```

See `sample_orders.csv` for example data.

## 🛠️ Development

```bash
# Build binary
go build -o oms-server cmd/server/main.go

# Run tests
go test ./...

# Clean build
make clean && make build
```

## 🐳 Docker

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

## 📋 Dependencies

- **Go 1.22+**
- **Docker & Docker Compose**
- **go_commons** (Omniful internal library)

## 🎯 Production Ready

- ✅ Real Kafka integration with go_commons
- ✅ MongoDB with indexes and aggregation
- ✅ Batch processing for large files
- ✅ Comprehensive error handling
- ✅ Invalid record logging and recovery
- ✅ Health checks and monitoring logs
- ✅ Graceful service degradation

## 🔄 Order Lifecycle

1. **Upload** → CSV file uploaded to S3
2. **Queue** → SQS message triggers processing
3. **Validate** → SKU/Hub validation via IMS
4. **Store** → Valid orders saved to MongoDB (`on_hold`)
5. **Events** → Kafka `order.created` events emitted
6. **Invalid** → Invalid records logged to downloadable CSV

## 🚀 Next Steps

- Implement Kafka consumer for order finalization
- Add inventory checking and atomic updates
- Enhance monitoring and metrics
- Add order status workflow management

---

**Ready to process millions of orders with confidence! 🎉**
