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

## Postman Collection

A Postman collection for **Auth APIs** is in `postman/MARVRON_Auth_APIs.postman_collection.json`. Import it in Postman; set `base_url` (e.g. `http://localhost:8080`). After **Login** or **Register**, the token is saved automatically for protected requests. See `postman/README.md` for details.

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

## Deploy to Railway

The app is configured for [Railway](https://railway.app). Use the `PORT`, `DATABASE_URL`, and `REDIS_URL` environment variables (Railway sets or provides these when you add Postgres/Redis).

1. Push this repo to GitHub and connect it to a new Railway project.
2. Add **PostgreSQL** and optionally **Redis** from the Railway dashboard.
3. Set `JWT_SECRET`, `QR_ENCRYPTION_KEY`, `QR_SIGNATURE_SECRET`, and `ENVIRONMENT=production` in the service variables.
4. Deploy; Railway uses `railway.toml` for build and start commands.

See **[RAILWAY.md](RAILWAY.md)** for step-by-step deployment instructions.

## Deploy to AWS

Full guide: **[docs/AWS_DEPLOY.md](docs/AWS_DEPLOY.md)** (Lightsail, EC2, RDS, App Runner / ECS overview).

### EC2 (quick)

1. **EC2** → Launch instance → **Ubuntu 24.04 LTS** → instance type (e.g. `t3.micro`) → **key pair** (download `.pem` to your Mac; your IAM user needs `ec2:CreateKeyPair` or use an **existing** key).
2. **Security group**: allow **SSH (22)** from your IP; allow **Custom TCP 8080** for the API (or restrict by IP).
3. Launch → copy **Public IPv4**.
4. SSH: `ssh -i /path/to/key.pem ubuntu@YOUR_PUBLIC_IP` (user is `ubuntu` on the official Ubuntu AMI).
5. On the instance: install **Docker** + **git**, clone this repo, run **Postgres** (`docker compose up -d postgres`) and the API container — **copy-paste commands** are in [docs/AWS_DEPLOY.md](docs/AWS_DEPLOY.md) (**EC2** section).
6. Optional: **Elastic IP** for a fixed public address; **RDS PostgreSQL** instead of Docker Postgres for production.

**Health check:** `GET http://YOUR_PUBLIC_IP:8080/health`

**Live EC2 API (current deploy):** base URL `http://98.92.77.116:8080` — health: `GET http://98.92.77.116:8080/health`. Change this if the instance public IP changes (unless you use an Elastic IP).

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
