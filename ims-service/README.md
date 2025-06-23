# IMS (Inventory Management System) Service

A microservice for managing inventory, hubs, and SKUs with support for multi-tenancy.

## Features

- **Hub Management**: CRUD operations for hubs
- **SKU Management**: CRUD operations for SKUs
- **Inventory Management**:
  - Atomic upsert of inventory levels
  - View inventory with filtering by hub, seller, and SKU codes
  - Support for available, reserved, and in-transit quantities

## Tech Stack

- **Go** for the backend
- **PostgreSQL** for data storage
- **Redis** for caching
- **Go Commons** for common utilities and server setup
- **Chi** for HTTP routing
- **GORM** for ORM

## Getting Started

### Prerequisites

- Go 1.16+
- PostgreSQL 12+
- Redis 6+

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/ims-service.git
   cd ims-service
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Set up environment variables:
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. Run database migrations:
   ```bash
   # Install migrate tool if you haven't
   go get -u github.com/golang-migrate/migrate/v4/cmd/migrate
   
   # Run migrations
   make migrate-up
   ```

5. Start the server:
   ```bash
   make run
   ```

## API Documentation

API documentation is available in the `docs` directory and can be viewed using Swagger UI.

### Endpoints

#### Inventory

- `POST /api/v1/inventory` - Update or insert inventory
- `GET /api/v1/inventory` - Get inventory with filters

## Environment Variables

```env
# Server
ENV=development
SERVER_PORT=:8080
GRACEFUL_SHUTDOWN_TIMEOUT=10s

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=ims
DB_SSLMODE=disable

# Redis
REDIS_ADDRESS=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
```

## Running Tests

```bash
make test
```

## Deployment

```bash
make build
./bin/ims-service
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
