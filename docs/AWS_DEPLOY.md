# Deploy MARVRON on AWS

We can’t log into your AWS account for you. Pick **one** path below. **Easiest first.**

---

## Simplest: AWS Lightsail (small server + optional managed DB)

Same idea as any VPS: one Linux box, Docker, done. Fewer AWS services than App Runner + RDS.

### 1) Create a Lightsail instance

1. Open [AWS Lightsail](https://lightsail.aws.amazon.com/).
2. **Create instance** → **Linux/Unix** → **Ubuntu 24.04**.
3. Pick the **$5/month** (or higher) plan.
4. Name it e.g. `marrvaron-api` → **Create instance**.

### 2) Open port 8080

1. Click your instance → **Networking** tab.
2. **Add rule** → **Application** custom → **TCP** `8080` → Save.  
   (SSH `22` is usually already open.)

### 3) Connect (browser or SSH)

Use **Connect using SSH** in Lightsail, or SSH from your Mac with the key Lightsail gives you.

### 4) Install Docker and clone the repo

```bash
sudo apt update && sudo apt install -y docker.io git
sudo usermod -aG docker ubuntu    # if your user is ubuntu; then log out and back in
# or run docker with sudo for now

sudo git clone https://github.com/Mehboob-alam1/marrvaron.git /opt/marrvaron
cd /opt/marrvaron
```

### 5) Database — pick one

**A) Postgres in Docker on the same instance (quickest)**  

```bash
sudo docker compose up -d postgres
# wait ~15s
sudo docker build -t marrvaron-api .
NET=$(sudo docker inspect marvaron_postgres -f '{{range $k,$v := .NetworkSettings.Networks}}{{$k}}{{end}}')
sudo docker rm -f marrvaron-api 2>/dev/null
sudo docker run -d --name marrvaron-api --restart unless-stopped --network "$NET" \
  -e ENVIRONMENT=production -e DB_HOST=postgres -e DB_PORT=5432 \
  -e DB_USER=marvaron_user -e DB_PASSWORD=marvaron_password -e DB_NAME=marvaron_db \
  -e DB_SSLMODE=disable \
  -e JWT_SECRET=replace-with-long-random-secret \
  -e 'QR_ENCRYPTION_KEY=exactly-32-characters-long-key!!' \
  -e QR_SIGNATURE_SECRET=replace-with-another-secret \
  -p 8080:8080 marrvaron-api
```

**B) Lightsail managed PostgreSQL (a bit more “AWS”, still simple)**  

1. Lightsail → **Databases** → **Create database** → PostgreSQL.  
2. Note **endpoint**, **user**, **password**, **database name**.  
3. In the database → **Networking** → allow your **Lightsail instance** as a trusted source.  
4. On the instance, run only the API container with **`DATABASE_URL`** (no local Postgres):

```bash
sudo docker build -t marrvaron-api /opt/marrvaron
sudo docker rm -f marrvaron-api 2>/dev/null
sudo docker run -d --name marrvaron-api --restart unless-stopped \
  -e ENVIRONMENT=production \
  -e DATABASE_URL='postgresql://USER:PASSWORD@ENDPOINT:5432/DBNAME?sslmode=require' \
  -e JWT_SECRET=replace-with-long-random-secret \
  -e 'QR_ENCRYPTION_KEY=exactly-32-characters-long-key!!' \
  -e QR_SIGNATURE_SECRET=replace-with-another-secret \
  -p 8080:8080 marrvaron-api
```

(URL-encode special characters in the password if needed.)

### 6) Test

Use your instance’s **public IP** from the Lightsail console:

```bash
curl http://YOUR_LIGHTSAIL_PUBLIC_IP:8080/health
```

API base: `http://YOUR_LIGHTSAIL_PUBLIC_IP:8080/api/v1/...`

**After code changes:** `cd /opt/marrvaron && sudo git pull && sudo docker build -t marrvaron-api .` then recreate the `marrvaron-api` container (same `docker run` as above).

---

## EC2 (same deployment as Lightsail, full AWS VM)

Use this if you want a standard **EC2** instance instead of Lightsail.

### 1) Launch an EC2 instance

1. AWS Console → **EC2** → **Launch instance**.
2. **Name**: e.g. `marrvaron-api`.
3. **AMI**: **Ubuntu Server 24.04 LTS** (64-bit x86).
4. **Instance type**: **t3.micro** (free tier eligible) or **t3.small** if you need more RAM.
5. **Key pair**: **Create new key pair** → type **RSA** → **.pem** → **Create** → **download** the `.pem` file to your Mac (you need it for SSH).

### 2) Network settings (security group)

Under **Network settings** → **Edit**:

- **Allow SSH traffic from**: **My IP** (recommended) so only you can SSH.
- **Allow HTTPS** / **Allow HTTP**: optional if you add a load balancer later.
- Click **Add security group rule**:
  - **Type**: Custom TCP  
  - **Port range**: `8080`  
  - **Source**: **Anywhere** `0.0.0.0/0` for testing, or **My IP** to restrict who can call the API.

Keep **SSH port 22** allowed from **My IP** or you will lock yourself out.

### 3) Storage

Default **8–20 GiB** gp3 is usually enough.

### 4) Launch

**Launch instance**. Wait until **Instance state** = **Running**. Copy the **Public IPv4 address**.

### 5) SSH from your Mac

```bash
chmod 400 ~/Downloads/your-key.pem    # path to the .pem you downloaded
ssh -i ~/Downloads/your-key.pem ubuntu@YOUR_PUBLIC_IP
```

On the official Ubuntu AMI the default user is **`ubuntu`** (not `root`).

### 6) Install Docker and deploy (same as Lightsail)

On the EC2 instance:

```bash
sudo apt update && sudo apt install -y docker.io git
sudo usermod -aG docker ubuntu
# log out and SSH back in so docker works without sudo, OR use sudo docker below

sudo git clone https://github.com/Mehboob-alam1/marrvaron.git /opt/marrvaron
cd /opt/marrvaron
```

**Database A — Postgres in Docker on this EC2 (quickest)**

```bash
sudo docker compose up -d postgres
sleep 15
sudo docker build -t marrvaron-api .
NET=$(sudo docker inspect marvaron_postgres -f '{{range $k,$v := .NetworkSettings.Networks}}{{$k}}{{end}}')
sudo docker rm -f marrvaron-api 2>/dev/null
sudo docker run -d --name marrvaron-api --restart unless-stopped --network "$NET" \
  -e ENVIRONMENT=production -e DB_HOST=postgres -e DB_PORT=5432 \
  -e DB_USER=marvaron_user -e DB_PASSWORD=marvaron_password -e DB_NAME=marvaron_db \
  -e DB_SSLMODE=disable \
  -e JWT_SECRET=replace-with-long-random-secret \
  -e 'QR_ENCRYPTION_KEY=exactly-32-characters-long-key!!' \
  -e QR_SIGNATURE_SECRET=replace-with-another-secret \
  -p 8080:8080 marrvaron-api
```

**Database B — Amazon RDS PostgreSQL**

1. Create **RDS** PostgreSQL in the **same VPC** as the EC2 (or use public RDS with strict security group).
2. RDS security group: inbound **5432** from the **EC2 instance’s security group**.
3. On EC2, run the API container with `-e DATABASE_URL='postgresql://...'` (same pattern as Lightsail section B).

### 7) Test

From your Mac:

```bash
curl http://YOUR_PUBLIC_IP:8080/health
```

### 8) Elastic IP (optional)

If you stop/start the instance, the **public IP can change**. In **EC2** → **Elastic IPs** → **Allocate** → **Associate** with your instance to get a **fixed** IP.

### 9) Updates after `git push`

```bash
ssh -i ~/Downloads/your-key.pem ubuntu@YOUR_PUBLIC_IP
cd /opt/marrvaron && sudo git pull && sudo docker build -t marrvaron-api .
sudo docker rm -f marrvaron-api
# run the same docker run ... command as in step 6 (with $NET if using local Postgres)
```

---

## More “managed” AWS (when you outgrow a single VM)

| If you want… | Use |
|--------------|-----|
| HTTPS URL + auto deploy from GitHub | **App Runner** + **RDS** (set `DATABASE_URL`, port `8080`, health `/health`) |
| Full control, auto scaling | **ECS Fargate** + **ALB** + **RDS** |

Those need VPC/security groups; follow AWS console wizards or hire DevOps when you’re ready.

---

## Environment variables (all paths)

| Variable | Required |
|----------|----------|
| `DATABASE_URL` **or** `DB_HOST` + `DB_USER` + `DB_PASSWORD` + `DB_NAME` | Yes |
| `JWT_SECRET` | Yes (production) |
| `QR_ENCRYPTION_KEY` | Yes |
| `QR_SIGNATURE_SECRET` | Yes |
| `REDIS_URL` | Optional (OTP works with Postgres) |

---

## Repo files

- **`Dockerfile`** — build for Lightsail / EC2 / App Runner / ECS.
- **`docs/VPS_BACKEND_SETUP_TEMPLATE.txt`** — generic VPS checklist (same steps as Lightsail).

---

## What we can’t do

Create resources in **your** AWS account, store passwords, or click the console for you. Use the steps above in Lightsail (simplest) or upgrade to App Runner/ECS later.
