# Postman – MARVRON Auth APIs

## Import

1. Open Postman.
2. **Import** → **Upload Files** → select `MARVRON_Auth_APIs.postman_collection.json`.
3. The collection **MARVRON - Auth APIs** will appear.

## Variables

- **base_url**: API base URL (default: `http://localhost:8080`). For Railway use `https://your-app.up.railway.app`.
- **token**: Set automatically after **Login** or **Register** (used as Bearer token for protected requests). You can also set it manually.

To change `base_url`: Collection → **Variables** → set **Current value** for `base_url`.

## Flow

1. **Health Check** – Check server is up (`GET /health`).
2. **Register** – Create a user (e.g. `customer`, `distributor`, `courier`). The returned `token` is stored in the collection.
3. **Login** – Get a new token (overwrites `token`).
4. **Get Profile** – Uses stored token (Bearer).
5. **Update Profile** – Uses stored token.
6. **Close Account** – Uses stored token (deactivates account).

**OTP (optional):** **Send OTP** then **Verify OTP**. Requires Redis and a valid user email/phone. Verify OTP returns a JWT and saves it to `token`.

## Auth endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/auth/register` | No | Register (roles: customer, distributor, courier) |
| POST | `/api/v1/auth/login` | No | Login; returns JWT |
| POST | `/api/v1/auth/otp/send` | No | Send OTP to email/phone |
| POST | `/api/v1/auth/otp/verify` | No | Verify OTP; returns JWT if user exists |
| GET | `/api/v1/auth/profile` | Bearer | Get current user profile |
| PUT | `/api/v1/auth/profile` | Bearer | Update profile |
| DELETE | `/api/v1/auth/account` | Bearer | Deactivate account |

Protected requests use **Authorization: Bearer {{token}}** (collection variable).
