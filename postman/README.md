# Postman – MARVRON Auth APIs

## For frontend developers (copy-paste)

| Use | Value |
|-----|--------|
| **API base URL** | `http://98.92.77.116:8080` |
| **Health** | `GET /health` → full URL: `http://98.92.77.116:8080/health` |
| **Auth API prefix** | `/api/v1/auth` |

Examples (append to base, no double slashes):

- Register: `POST http://98.92.77.116:8080/api/v1/auth/register`
- Login: `POST http://98.92.77.116:8080/api/v1/auth/login`
- Profile: `GET http://98.92.77.116:8080/api/v1/auth/profile` (Bearer JWT)

**CORS:** If the browser blocks requests, the backend must allow your frontend origin (configure CORS on the server for production).

Import **`MARVRON_Auth_APIs.postman_collection.json`** — collection variable **`base_url`** is already set to the URL above.

## Import

1. Open Postman.
2. **Import** → **Upload Files** → select `MARVRON_Auth_APIs.postman_collection.json`.
3. The collection **MARVRON - Auth APIs** will appear.

## Variables

- **base_url**: API base URL (default in the JSON file: `http://98.92.77.116:8080` on AWS EC2). For local use `http://localhost:8080`. For Railway use `https://your-app.up.railway.app`.
- **token**: JWT — auto-saved after **Login**, legacy **Register**, **Verify phone (complete)**, **Verify OTP** (when allowed), **Enable role**, **Switch active role**.
- **registration_token**: Auto-saved after **1 Register profile** and refreshed after **2 Register password** (multi-step signup).

To change `base_url`: Collection → **Variables** → set **Current value** for `base_url`.

## Collection folders (in Postman)

1. **01 Registration (multi-step)** – profile → password → verify phone.
2. **02 Session** – legacy register + login.
3. **03 OTP** – send / verify (general).
4. **04 Profile & account** – get/update profile, close account (Bearer `token`).
5. **05 Roles** – enable distributor/courier/vendor, switch active role.
6. **06 Health** – `GET /health`.

## Flow

1. **Health Check** – `GET /health`.
2. **Multi-step:** run folder **01** in order, or **02 → Register (legacy)** for one-shot signup.
3. **Login** – email + password (needs completed registration for multi-step users).
4. **Profile** / **Roles** – use **token** as collection Bearer.

**OTP:** Redis if configured, else Postgres. Dev: OTP appears in **server logs**.

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
