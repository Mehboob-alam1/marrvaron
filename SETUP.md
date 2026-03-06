# Setup Guide - MARVRON Backend

## Prerequisiti

1. **Go 1.21 o superiore**
   - Scarica da: https://golang.org/dl/
   - Verifica installazione: `go version`

2. **PostgreSQL 14+**
   - Installazione: https://www.postgresql.org/download/

3. **Redis 7+**
   - Installazione: https://redis.io/download

4. **Docker e Docker Compose** (opzionale, per sviluppo locale)
   - Installazione: https://www.docker.com/get-started

5. **Apache Kafka** (opzionale, per eventi in tempo reale)
   - Può essere avviato tramite Docker Compose

## Installazione

### 1. Clona/Scarica il progetto

```bash
cd /Users/mac/Documents/marvaron
```

### 2. Installa le dipendenze

```bash
go mod download
go mod tidy
```

### 3. Configura le variabili d'ambiente

Copia il file `.env.example` in `.env`:

```bash
cp .env.example .env
```

Modifica `.env` con le tue configurazioni:

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=marvaron_user
DB_PASSWORD=marvaron_password
DB_NAME=marvaron_db

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# JWT - IMPORTANTE: Cambia in produzione!
JWT_SECRET=your-super-secret-jwt-key-change-in-production

# QR Code Encryption - IMPORTANTE: Cambia in produzione!
QR_ENCRYPTION_KEY=your-32-byte-encryption-key-for-aes-256
QR_SIGNATURE_SECRET=your-signature-secret-for-qr-verification
```

### 4. Avvia i servizi con Docker Compose (consigliato)

```bash
docker-compose up -d
```

Questo avvierà:
- PostgreSQL sulla porta 5432
- Redis sulla porta 6379
- Zookeeper sulla porta 2181
- Kafka sulla porta 9092

### 5. Oppure configura manualmente i servizi

#### PostgreSQL
```bash
# Crea database
createdb marvaron_db

# Oppure tramite psql
psql -U postgres
CREATE DATABASE marvaron_db;
CREATE USER marvaron_user WITH PASSWORD 'marvaron_password';
GRANT ALL PRIVILEGES ON DATABASE marvaron_db TO marvaron_user;
```

#### Redis
```bash
redis-server
```

### 6. Esegui le migrations

Le migrations vengono eseguite automaticamente all'avvio del server tramite GORM AutoMigrate.

### 7. Avvia il server

```bash
# Sviluppo
go run ./cmd/server

# Oppure build e esegui
go build -o bin/server ./cmd/server
./bin/server
```

Il server sarà disponibile su `http://localhost:8080`

## Credenziali Super Admin di Default

Alla prima esecuzione, viene creato automaticamente un super admin:

- **Email**: `admin@marvaron.com`
- **Password**: `admin123`

**⚠️ IMPORTANTE**: Cambia immediatamente la password in produzione!

## Test dell'API

### Health Check
```bash
curl http://localhost:8080/health
```

### Registrazione
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "first_name": "Test",
    "last_name": "User",
    "role": "customer"
  }'
```

### Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

## Struttura del Progetto

```
marvaron/
├── cmd/
│   └── server/
│       └── main.go          # Entry point principale
├── internal/
│   ├── config/              # Configurazione
│   ├── models/              # Modelli database (GORM)
│   ├── handlers/            # HTTP handlers (Gin)
│   ├── services/            # Business logic (da implementare)
│   ├── middleware/          # Middleware (auth, CORS, rate limit)
│   ├── repository/          # Data access layer (da implementare)
│   ├── utils/               # Utilities (JWT, OTP, QR, password)
│   ├── database/            # Connessioni DB e Redis
│   └── kafka/               # Kafka producers/consumers
├── pkg/                     # Package riutilizzabili (opzionale)
├── migrations/              # Database migrations (opzionale)
├── docker-compose.yml       # Setup servizi locali
├── Dockerfile              # Container Docker
├── go.mod                  # Dipendenze Go
└── README.md              # Documentazione principale
```

## Sviluppo

### Formattazione codice
```bash
go fmt ./...
```

### Test
```bash
go test ./...
```

### Build per produzione
```bash
go build -o bin/server ./cmd/server
```

## Docker

### Build immagine
```bash
docker build -t marvaron-backend .
```

### Esegui container
```bash
docker run -p 8080:8080 --env-file .env marvaron-backend
```

## Note Importanti

1. **Sicurezza**:
   - Cambia `JWT_SECRET` e `QR_ENCRYPTION_KEY` in produzione
   - Usa HTTPS in produzione
   - Configura firewall appropriato

2. **Database**:
   - Le migrations sono automatiche tramite GORM
   - Per produzione, considera migrazioni manuali più controllate

3. **Kafka**:
   - Attualmente commentato nel main.go
   - Decommentare quando necessario

4. **Redis**:
   - Usato per cache, OTP storage, rate limiting
   - Il server funziona anche senza Redis (con limitazioni)

5. **Performance**:
   - Considera connection pooling per PostgreSQL
   - Implementa caching strategico con Redis
   - Usa CDN per asset statici

## Troubleshooting

### Errore connessione database
- Verifica che PostgreSQL sia in esecuzione
- Controlla credenziali in `.env`
- Verifica che il database esista

### Errore connessione Redis
- Verifica che Redis sia in esecuzione
- Il server continuerà a funzionare senza Redis (con warning)

### Porta già in uso
- Cambia `SERVER_PORT` in `.env`
- Oppure termina il processo che usa la porta 8080

## Prossimi Passi

1. Implementare servizi di notifica (Email, SMS, Push)
2. Integrare gateway di pagamento
3. Implementare Kafka consumers per eventi
4. Aggiungere test unitari e di integrazione
5. Configurare CI/CD
6. Setup monitoring e logging (Prometheus, Grafana)
7. Implementare backup automatici database
