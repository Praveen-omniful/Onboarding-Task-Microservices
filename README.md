# Microservices Project - OMS & IMS

A complete microservices architecture implementing Order Management Service (OMS) and Inventory Management Service (IMS) with full integration support for Kafka, MongoDB, PostgreSQL, Redis, and AWS services.

## 🚀 Architecture Overview

This project consists of two main microservices:

- **OMS (Order Management Service)** - Handles order processing, CSV uploads, and data pipeline
- **IMS (Inventory Management Service)** - Manages inventory, SKUs, and hub operations

## 📋 Prerequisites

- Docker & Docker Compose
- Go 1.19+
- Git

## 🛠️ Quick Start

### 1. Clone the Repository
```bash
git clone https://github.com/praveen-omniful/microservices-project.git
cd microservices-project
```

### 2. Start Infrastructure Services
```bash
# Start MongoDB, PostgreSQL, Redis, Kafka, and LocalStack
docker-compose up -d
```

### 3. Run Database Migrations (IMS)
```bash
cd ims-service
go run cmd/migrate/main.go
```

### 4. Start Services

**Start IMS Service:**
```bash
cd ims-service
go run cmd/server/main.go
# Runs on http://localhost:8081
```

**Start OMS Service:**
```bash
cd oms-service
go run cmd/server/main.go
# Runs on http://localhost:8084
```

## 🔧 Service Details

### OMS (Order Management Service)
- **Port:** 8084
- **Health Check:** `GET /health`
- **Upload Endpoint:** `POST /upload`
- **Statistics:** `GET /stats`

**Features:**
- CSV file upload and processing
- Kafka integration for message streaming
- MongoDB storage for processed orders
- S3 integration with LocalStack
- SQS integration for queue management
- Data validation and error handling

### IMS (Inventory Management Service)
- **Port:** 8081
- **Health Check:** `GET /health`
- **API Base:** `/api/v1/`

**Endpoints:**
- **SKUs:** `GET/POST /api/v1/skus`
- **Inventory:** `GET/POST /api/v1/inventory`
- **Hubs:** `GET/POST /api/v1/hubs`

## 📊 Testing the System

### Manual Testing

1. **Health Checks:**
```bash
curl http://localhost:8081/health  # IMS
curl http://localhost:8084/health  # OMS
```

2. **Upload Test Orders:**
```bash
curl -X POST -F "file=@test_orders.csv" http://localhost:8084/upload
```

3. **Check Statistics:**
```bash
curl http://localhost:8084/stats
```

### Sample Data

The project includes sample CSV files:
- `test_orders.csv` - Small test dataset
- `bulk_test_1k.csv` - 1000 orders for load testing

## 🗄️ Database Schema

### PostgreSQL (IMS)
- **skus** - Product information
- **inventory** - Stock levels
- **hubs** - Distribution centers

### MongoDB (OMS)
- **Collection:** `orders`
- **Database:** `oms`

## 🔗 Integration Services

### Kafka
- **Brokers:** localhost:9092
- **Topics:** Order processing events

### MongoDB
- **Host:** localhost:27017
- **Database:** oms

### PostgreSQL
- **Host:** localhost:5433
- **Database:** IMS
- **User:** postgres

### LocalStack (AWS Services)
- **Endpoint:** http://localhost:4566
- **Services:** S3, SQS

## 📈 Monitoring & Health

Run the project status script to check all services:
```bash
./project_status.sh
```

## 🏗️ Project Structure

```
microservices-project/
├── ims-service/
│   ├── cmd/
│   │   ├── migrate/         # Database migrations
│   │   └── server/          # IMS server
│   ├── internal/
│   │   ├── api/handlers/    # REST API handlers
│   │   ├── config/          # Configuration
│   │   ├── models/          # Data models
│   │   ├── repository/      # Data access layer
│   │   └── service/         # Business logic
│   ├── migrations/          # SQL migration files
│   └── go.mod              # Go dependencies
│
├── oms-service/
│   ├── cmd/server/          # OMS server
│   ├── internal/
│   │   ├── ims/            # IMS integration
│   │   ├── kafka/          # Kafka integration
│   │   ├── mongodb/        # MongoDB client
│   │   ├── processor/      # CSV processing
│   │   └── s3/             # S3 integration
│   ├── docker-compose.yml  # Infrastructure setup
│   └── go.mod              # Go dependencies
│
├── project_status.sh        # Health check script
└── README.md               # This file
```

## 🔧 Configuration

### Environment Variables

Both services use `.env` files for configuration. Key variables:

**IMS Service:**
- `DB_HOST`, `DB_PORT`, `DB_NAME`
- `DB_USER`, `DB_PASSWORD`

**OMS Service:**
- `MONGODB_URI`
- `KAFKA_BROKERS`
- `AWS_ENDPOINT` (LocalStack)

## 🎯 Production Readiness

✅ **Completed Features:**
- Full microservices architecture
- Database migrations
- Health monitoring
- Error handling and validation
- Integration with external services
- Comprehensive logging
- Clean code structure

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## 📝 License

This project is licensed under the MIT License.

## 👥 Team

Developed by Praveen Omniful

---

**🎉 Ready for Production!** 

This microservices project demonstrates enterprise-level architecture with full integration capabilities and production-ready features.
