# MARVRON Authentication & Sales App - Backend

Golang backend for the MARVRON application that handles product authentication, inventory, sales, distributors, customers, and couriers.

## Key Features

- 🔐 Secure authentication with JWT and OTP
- 📱 Product verification via encrypted QR codes
- 📦 Advanced inventory management
- 🛒 Cart and checkout system
- 👥 Multi-role management (Admin, Distributor, Customer, Courier)
- 📊 Full admin dashboard
- 🔄 Kafka integration for real-time events
- ⚡ Redis cache for optimal performance

## Tech Stack

- **Backend**: Golang 1.21+
- **Database**: PostgreSQL
- **Cache**: Redis
- **Message Queue**: Apache Kafka
- **Web Framework**: Gin

## Project Structure

```
marvaron/
├── cmd/
│   └── server/
│       └── main.go          # Entry point
├── internal/
│   ├── config/              # Configuration
│   ├── models/              # Database models
│   ├── handlers/            # HTTP handlers
│   ├── services/            # Business logic
│   ├── middleware/          # Middleware (auth, logging, etc.)
│   ├── repository/         # Data access layer
│   ├── utils/               # Utilities (QR, encryption, etc.)
│   └── kafka/               # Kafka producers/consumers
├── pkg/                     # Reusable packages
├── migrations/              # Database migrations
├── docker-compose.yml       # Local setup
└── Dockerfile               # Docker container
```

## Local Setup

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 14+
- Redis 7+
- Apache Kafka (optional for local development)

### Installation

1. Clone the repository
2. Copy `.env.example` to `.env` and configure the variables
3. Install dependencies:
```bash
go mod download
```

4. Start services with Docker Compose:
```bash
docker-compose up -d
```

5. Run migrations (migrations run automatically on server start via GORM AutoMigrate)

6. Start the server:
```bash
go run ./cmd/server/main.go
```

## API Endpoints

### Authentication
- `POST /api/v1/auth/register` - User registration
- `POST /api/v1/auth/login` - Login
- `POST /api/v1/auth/otp/send` - Send OTP
- `POST /api/v1/auth/otp/verify` - Verify OTP

### QR Code
- `POST /api/v1/qr/scan` - Scan QR code
- `GET /api/v1/qr/verify/:token` - Verify QR token
- `GET /api/v1/qr/history` - Scan history

### Products
- `GET /api/v1/products` - List products
- `GET /api/v1/products/:id` - Product details
- `POST /api/v1/products` - Create product (Admin)
- `PUT /api/v1/products/:id` - Update product (Admin)

### Orders
- `POST /api/v1/orders` - Create order
- `GET /api/v1/orders` - List orders
- `GET /api/v1/orders/:id` - Order details
- `PUT /api/v1/orders/:id` - Update order

### Admin
- `GET /api/v1/admin/dashboard` - Admin dashboard
- `POST /api/v1/admin/admins` - Create admin user (Super Admin)
- `GET /api/v1/admin/analytics` - Analytics

## Development

### Run tests
```bash
go test ./...
```

### Build for production
```bash
go build -o bin/server ./cmd/server/main.go
```

## License

Proprietary - MARVRON
