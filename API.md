# MARVRON API Documentation

## Base URL
```
http://localhost:8080/api/v1
```

## Autenticazione

La maggior parte degli endpoint richiede autenticazione tramite JWT token nell'header:
```
Authorization: Bearer <token>
```

## Endpoints

### Autenticazione

#### Registrazione
```http
POST /auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123",
  "phone": "+1234567890",
  "first_name": "John",
  "last_name": "Doe",
  "role": "customer", // customer, distributor, courier
  "marketing_opt_in": true
}
```

#### Login
```http
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}
```

#### Invia OTP
```http
POST /auth/otp/send
Content-Type: application/json

{
  "identifier": "user@example.com",
  "method": "email" // email o sms
}
```

#### Verifica OTP
```http
POST /auth/otp/verify
Content-Type: application/json

{
  "identifier": "user@example.com",
  "otp": "123456"
}
```

#### Profilo Utente
```http
GET /auth/profile
Authorization: Bearer <token>
```

#### Aggiorna Profilo
```http
PUT /auth/profile
Authorization: Bearer <token>
Content-Type: application/json

{
  "first_name": "John",
  "last_name": "Doe",
  "phone": "+1234567890",
  "marketing_opt_in": true
}
```

### QR Code

#### Scansiona QR Code
```http
POST /qr/scan
Content-Type: application/json

{
  "encrypted_token": "...",
  "signature": "...",
  "device_id": "device123",
  "device_type": "ios",
  "location_lat": 40.7128,
  "location_lng": -74.0060,
  "location_address": "New York, NY",
  "scan_method": "camera"
}
```

#### Verifica QR Code
```http
GET /qr/verify/:token
```

#### Storico Scansioni
```http
GET /qr/history
Authorization: Bearer <token>
```

#### Genera QR Code (Admin)
```http
POST /qr/generate
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "product_id": "uuid",
  "batch_number": "BATCH001",
  "serial_number": "SERIAL001",
  "inventory_id": "uuid",
  "display_info": "{\"custom\": \"info\"}"
}
```

### Prodotti

#### Lista Prodotti
```http
GET /products?page=1&limit=20&category=electronics&search=phone
```

#### Dettaglio Prodotto
```http
GET /products/:id
```

#### Crea Prodotto (Admin)
```http
POST /products
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "name": "Product Name",
  "description": "Product description",
  "sku": "SKU123",
  "barcode": "123456789",
  "category": "electronics",
  "brand": "Brand Name",
  "base_price": 99.99,
  "currency": "USD",
  "image_url": "https://...",
  "weight": 0.5,
  "dimensions": "10x10x5",
  "is_authenticatable": true
}
```

#### Aggiorna Prodotto (Admin)
```http
PUT /products/:id
Authorization: Bearer <admin_token>
```

#### Aggiungi Item Inventario (Admin)
```http
POST /products/inventory
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "product_id": "uuid",
  "batch_number": "BATCH001",
  "serial_number": "SERIAL001",
  "quantity": 100,
  "cost_price": 50.00,
  "location": "Warehouse A"
}
```

### Ordini

#### Crea Ordine
```http
POST /orders
Content-Type: application/json

{
  "items": [
    {
      "product_id": "uuid",
      "qr_code_id": "uuid", // opzionale
      "quantity": 2
    }
  ],
  "shipping_address": "123 Main St",
  "billing_address": "123 Main St",
  "payment_method": "card",
  "notes": "Handle with care",
  "save_for_later": false
}
```

#### Lista Ordini
```http
GET /orders
Authorization: Bearer <token>
```

#### Dettaglio Ordine
```http
GET /orders/:id
Authorization: Bearer <token>
```

#### Aggiorna Ordine
```http
PUT /orders/:id
Authorization: Bearer <token>
Content-Type: application/json

{
  "status": "shipped",
  "payment_status": "paid",
  "notes": "Updated notes"
}
```

### Carrello

#### Aggiungi al Carrello
```http
POST /cart
Content-Type: application/json
X-Session-ID: <session_id> // Per utenti guest

{
  "product_id": "uuid",
  "qr_code_id": "uuid", // opzionale
  "quantity": 1
}
```

#### Visualizza Carrello
```http
GET /cart
X-Session-ID: <session_id> // Per utenti guest
```

#### Rimuovi dal Carrello
```http
DELETE /cart/:id
```

### Distributore

#### Informazioni Distributore
```http
GET /distributor/info
Authorization: Bearer <distributor_token>
```

#### Aggiorna Info Distributore
```http
PUT /distributor/info
Authorization: Bearer <distributor_token>
```

#### Richiedi Preventivo
```http
POST /distributor/price-quote
Authorization: Bearer <distributor_token>
Content-Type: application/json

{
  "product_id": "uuid",
  "quantity": 100,
  "requested_price": 80.00,
  "notes": "Bulk order"
}
```

#### Lista Preventivi
```http
GET /distributor/price-quotes
Authorization: Bearer <distributor_token>
```

### Admin

#### Dashboard
```http
GET /admin/dashboard
Authorization: Bearer <admin_token>
```

#### Analytics
```http
GET /admin/analytics?days=30
Authorization: Bearer <admin_token>
```

#### Crea Admin (Super Admin)
```http
POST /admin/admins
Authorization: Bearer <super_admin_token>
Content-Type: application/json

{
  "email": "admin@example.com",
  "password": "password123",
  "first_name": "Admin",
  "last_name": "User",
  "permissions": {
    "can_update_inventory": true,
    "can_generate_qr": true,
    "can_manage_orders": true,
    "can_manage_users": false,
    "can_send_promotions": true,
    "can_view_raw_store": true,
    "can_edit_raw_store": false
  }
}
```

#### Approva Distributore
```http
POST /admin/distributors/:id/approve
Authorization: Bearer <admin_token>
```

#### Badge QR Code
```http
POST /admin/qr/badge
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "qr_code_id": "uuid",
  "distributor_id": "uuid",
  "region_id": "uuid",
  "region_name": "North Region"
}
```

#### Lista Preventivi
```http
GET /admin/price-quotes?status=pending
Authorization: Bearer <admin_token>
```

#### Aggiorna Preventivo
```http
PUT /admin/price-quotes/:id
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "quoted_price": 85.00,
  "status": "approved",
  "notes": "Special pricing approved"
}
```

## Codici di Stato

- `200` - Successo
- `201` - Creato
- `400` - Richiesta non valida
- `401` - Non autorizzato
- `403` - Accesso negato
- `404` - Non trovato
- `409` - Conflitto
- `429` - Troppe richieste (Rate limit)
- `500` - Errore server

## Errori

Tutti gli errori seguono questo formato:
```json
{
  "error": "Messaggio di errore"
}
```
