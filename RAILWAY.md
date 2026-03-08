# Deploy MARVRON Backend to Railway

This guide walks you through deploying the MARVRON Go backend to [Railway](https://railway.app).

## Prerequisites

- A [Railway](https://railway.app) account (GitHub login supported)
- This repository pushed to GitHub

## Step 1: Create a new project on Railway

1. Go to [railway.app](https://railway.app) and sign in.
2. Click **New Project**.
3. Choose **Deploy from GitHub repo**.
4. Select your repository: `Mehboob-alam1/marrvaron` (or your fork).
5. Railway will detect the Go app and use the `railway.toml` in the repo.

## Step 2: Add PostgreSQL

1. In your project, click **+ New**.
2. Select **Database** → **PostgreSQL**.
3. Railway creates a Postgres service and injects **`DATABASE_URL`** into your app (no extra config needed if you link the service).

## Step 3: Add Redis (optional but recommended)

1. Click **+ New** again.
2. Select **Database** → **Redis**.
3. Railway creates a Redis service and injects **`REDIS_URL`** when linked.

## Step 4: Link services and set variables

1. Click your **backend service** (the one from the repo).
2. Go to **Variables**.
3. Ensure **DATABASE_URL** and **REDIS_URL** are present (they are added automatically when you add Postgres/Redis and link them in the same project; you can also add references from the Data tab).
4. Add these variables manually if not using add-ons or for overrides:

| Variable | Required | Description |
|----------|----------|-------------|
| `PORT` | Set by Railway | Do not set; Railway sets this. |
| `DATABASE_URL` | Yes (if using Postgres add-on) | Set automatically when you add Postgres and reference it. |
| `REDIS_URL` | No (optional) | Set automatically when you add Redis. App works without Redis (OTP and rate limit disabled). |
| `JWT_SECRET` | Yes | Strong secret for JWT (e.g. generate with `openssl rand -base64 32`). |
| `QR_ENCRYPTION_KEY` | Yes | 32-byte key for QR encryption (e.g. `openssl rand -base64 32`). |
| `QR_SIGNATURE_SECRET` | Yes | Secret for QR signature verification. |
| `ENVIRONMENT` | Recommended | Set to `production`. |

### Referencing Postgres/Redis in the same project

- In the backend service **Variables** tab, click **Add Variable** → **Add Reference**.
- Choose the variable from the Postgres or Redis service (e.g. `DATABASE_URL`, `REDIS_URL`).

## Step 5: Deploy

1. Railway builds the app using `railway.toml`:
   - **Build:** `go build -o server ./cmd/server`
   - **Start:** `./server`
2. The app listens on `PORT` (set by Railway).
3. After deploy, open the **Settings** of the backend service and use **Generate Domain** under **Networking** to get a public URL (e.g. `https://your-app.up.railway.app`).

## Step 6: Verify

- Open `https://your-app.up.railway.app/health` — you should see `{"status":"ok"}`.
- API base: `https://your-app.up.railway.app/api/v1`.

## Environment variables summary

| Variable | Set by | Notes |
|----------|--------|--------|
| `PORT` | Railway | Server port. |
| `DATABASE_URL` | You (or Postgres add-on) | Postgres connection string. |
| `REDIS_URL` | You (or Redis add-on) | Redis URL. Optional. |
| `JWT_SECRET` | You | Required in production. |
| `QR_ENCRYPTION_KEY` | You | 32-byte key. |
| `QR_SIGNATURE_SECRET` | You | For QR verification. |
| `ENVIRONMENT` | You | Use `production` on Railway. |

## Troubleshooting

- **Build fails:** Ensure `go.mod` is at repo root and `railway.toml` build command is `go build -o server ./cmd/server`.
- **App crashes on start:** Check **Logs** in Railway. Often due to missing `DATABASE_URL` or invalid `JWT_SECRET` in production.
- **502 Bad Gateway:** App may be listening on wrong port. The app uses `PORT` from the environment; do not set `SERVER_PORT` on Railway.
- **DB connection refused:** Confirm Postgres is in the same project and `DATABASE_URL` is referenced in the backend service variables.

## Custom domain (optional)

In the backend service → **Settings** → **Networking**, add a custom domain and point your DNS to the provided CNAME.
