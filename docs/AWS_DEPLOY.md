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

**AMI vs SSH user:** **Ubuntu** → log in as **`ubuntu`**. **Amazon Linux 2023** → log in as **`ec2-user`**. The checklist below is for **Amazon Linux 2023** (common default AMIs). For Ubuntu, skip to [§1 Launch (Ubuntu)](#1-launch-an-ec2-instance) and use `apt` as shown later.

---

### EC2 from scratch — Amazon Linux 2023 (full server setup)

Do these in order. If **`curl http://YOUR_PUBLIC_IP:8080/health`** from your Mac **times out**, the API is usually fine — **inbound TCP 8080** is missing on the **instance security group** (AWS drops the packet; the client sees a timeout, not “connection refused”).

#### A1) Launch the instance (AWS Console)

1. **EC2** → **Launch instance**.
2. **Name:** e.g. `marrvaron-api`.
3. **Application and OS Images:** **Amazon Linux** → **Amazon Linux 2023** (x86_64 or Arm — match the Docker Compose download in A5).
4. **Instance type:** **`t3.small`** or larger recommended (Docker `go build` is slow on `t3.micro`).
5. **Key pair:** Create or choose a key pair → download the **`.pem`** to your Mac.

#### A2) Network / security group (required for SSH + public API)

Under **Network settings** → **Edit**:

1. Prefer a **public subnet** and **Auto-assign public IP: Enable** (or attach an **Elastic IP** later).

2. **Create security group** (or select one you control) and set **inbound rules** as below. **What this is (detail):**
   - A **security group** is a **virtual firewall** attached to your instance’s **network interface** (in the VPC). It is **stateful**: if traffic is allowed **in**, the **reply** is allowed **out** automatically for that flow.
   - **Inbound rules** = “who on the internet may **open a connection to** which **ports** on this instance.” If there is **no** matching inbound rule, AWS **drops** those packets. Your laptop never gets a TCP “yes/no” from the app — the packet never reaches the OS — so the client often sees a **connection timeout** (not always “connection refused”).
   - **Outbound rules** default to **allow all** on new security groups; you rarely need to change them for this app.

3. **Inbound rules you need:**

| Type        | Port | Source        | Purpose |
|------------|------|---------------|---------|
| SSH        | 22   | **My IP**     | Only **your current public IP** may connect for administration (`ssh`). Reduces brute-force noise. If your home IP changes, add a new rule or switch to **your new IP**; otherwise SSH will time out too. |
| Custom TCP | 8080 | `0.0.0.0/0`   | **Any IPv4 address** may connect to port **8080** (your HTTP API). Needed so browsers, Postman, and phones can reach `http://YOUR_PUBLIC_IP:8080`. **`0.0.0.0/0`** means “the whole IPv4 internet” (testing). **If this rule is missing, `curl` and Chrome will hang and then fail with a timeout** — even if `curl http://127.0.0.1:8080/health` works **on the server**. |

4. **Why “My IP” for SSH but `0.0.0.0/0` for 8080?**
   - **Port 22 (SSH)** is constantly scanned on the internet. Limiting source to **My IP** means only your network can try to log in.
   - **Port 8080** is your **public API**. For a quick test you open it to **everyone** (`0.0.0.0/0`). For a **private** API, set source to **My IP** (same idea as SSH) or to a **load balancer** / **VPN** security group later.

5. **`0.0.0.0/0` in plain terms:** CIDR for “any IPv4 host.” It is **not** your instance IP — it means **clients may come from any IPv4 address**. (If you also use **IPv6**, you may need a separate inbound rule with source **`::/0`** for TCP 8080 on IPv6 — only if your instance has a public IPv6 and clients use it.)

6. **How to add or fix rules on an existing instance:** **EC2** → **Instances** → select instance → **Security** tab → click the **security group** name(s) → **Edit inbound rules** → **Add rule** → save. Changes apply in seconds; no reboot required.

7. **Common mistakes:** Rule exists on the **wrong** security group (not attached to this instance). **Typo** in port (**8008** vs **8080**). **Outbound** accidentally locked down (rare). Using **HTTPS** in the browser without TLS on the app (different error than timeout, but worth knowing).

8. **Timeout vs refused:** **Timeout** → often **security group / NACL / no route** (packet dropped). **Connection refused** → usually reached the host but **nothing listening** on that port, or a local firewall rejected it.

9. Launch the instance. Wait until **Running** and status checks pass. Copy **Public IPv4 address** (`YOUR_PUBLIC_IP`).

#### A3) SSH from your Mac (stay connected during long builds)

```bash
chmod 400 ~/Downloads/your-key.pem    # use your real .pem path
ssh -o ServerAliveInterval=60 -o ServerAliveCountMax=3 -i ~/Downloads/your-key.pem ec2-user@YOUR_PUBLIC_IP
```

Type **`yes`** if asked about host authenticity. Prompt should look like **`[ec2-user@ip-172-31-… ~]$`**.

#### A4) Install Docker and Git

```bash
sudo dnf update -y
sudo dnf install -y docker git
sudo systemctl enable --now docker
sudo usermod -aG docker ec2-user
```

Log out and SSH back in once so `docker` works without `sudo`, **or** keep using **`sudo docker`** for all Docker commands below.

#### A5) Install Docker Compose plugin (not in default AL2023 repos)

**x86_64** (most instances):

```bash
sudo mkdir -p /usr/local/lib/docker/cli-plugins
sudo curl -SL "https://github.com/docker/compose/releases/download/v2.29.7/docker-compose-linux-x86_64" -o /usr/local/lib/docker/cli-plugins/docker-compose
sudo chmod +x /usr/local/lib/docker/cli-plugins/docker-compose
```

**Arm (Graviton):** use `docker-compose-linux-aarch64` instead of `x86_64`.

Check:

```bash
sudo docker compose version
```

#### A6) Clone the repo and start Postgres

```bash
sudo git clone https://github.com/Mehboob-alam1/marrvaron.git /opt/marrvaron
cd /opt/marrvaron
sudo docker compose up -d postgres
sleep 20
sudo docker ps    # expect marvaron_postgres
```

#### A7) Build the API image (use `tmux` so SSH drops don’t kill the build)

```bash
sudo dnf install -y tmux
tmux new -s build
cd /opt/marrvaron
sudo docker build -t marrvaron-api .
```

Wait until the build finishes. Detach from tmux: **Ctrl+B** then **D**. Reattach: `tmux attach -t build`.

#### A8) Run the API container

Replace secrets with strong values. **`QR_ENCRYPTION_KEY`** must be **exactly 32 characters**.

```bash
cd /opt/marrvaron
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

**`-p 8080:8080`** publishes the app on the instance on **all interfaces** (`0.0.0.0:8080`).

#### A9) Verify on the server

```bash
sudo docker ps
sudo ss -tlnp | grep 8080
curl -s http://127.0.0.1:8080/health
```

Expect **`{"status":"ok"}`** and a listen line like **`0.0.0.0:8080`**.

#### A10) Verify from your Mac (needs A2 port 8080 rule)

```bash
curl -v http://YOUR_PUBLIC_IP:8080/health
```

Browser: **`http://YOUR_PUBLIC_IP:8080/health`** (use **http**, not **https**, unless you add TLS).

#### A11) If outside access still times out

1. Confirm **Public IPv4** in the console matches **`YOUR_PUBLIC_IP`**.  
2. **EC2** → instance → **Security** → security group → **Inbound rules** → **Custom TCP 8080** from `0.0.0.0/0` (test).  
3. On the server, **`ss`** must show **`0.0.0.0:8080`**, not only `127.0.0.1:8080`.  
4. Try **`curl` from a phone hotspot** to rule out your office/home blocking outbound 8080.

---

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

**What Step 5 means (detail)**

1. **`chmod 400 …your-key.pem`**  
   - Your key file is a **secret** (like a house key). SSH refuses to use it if the file is “too readable” (e.g. other users on your Mac could read it).  
   - `400` means: **only you** can read the file; nobody can write or execute it.  
   - Run this **once per key** (or again if you copy a new `.pem`). If you skip it, `ssh` may show *“UNPROTECTED PRIVATE KEY FILE”* and refuse to connect.  
   - Adjust the path: e.g. `~/Downloads/key_marvaron_aws.pem`.

2. **`ssh -i … ubuntu@YOUR_PUBLIC_IP`**  
   - **`ssh`** = “Secure Shell”: your Mac opens an **encrypted** connection to another computer (here, your EC2 instance) over the internet.  
   - **`-i ~/Downloads/your-key.pem`** = “use **this** private key to prove I am allowed to log in.” AWS put the matching **public** key on the server when you chose the key pair at launch. Only someone with the `.pem` can connect (plus: security group must allow your IP on port 22).  
   - **`ubuntu`** = the **username** on the server for the official Ubuntu image (not `root`, not your Mac username).  
   - **`YOUR_PUBLIC_IP`** = replace with the **Public IPv4 address** from the EC2 console (e.g. `3.91.200.10`). No `http://`, no port here — SSH default port is 22.

3. **“Like dialing a phone” (analogy)**  
   - Your Mac is the **phone**. The EC2 instance is the **person you call**.  
   - `chmod` = you make sure your “calling card” (key file) is stored safely before you call.  
   - `ssh` = you **dial** the server’s public IP; the server **answers** if the key matches and port 22 is open.  
   - After the call connects, **everything you type** in that window is sent to the server and runs **there** — that is why Step 6 happens in the “same” Terminal window.

4. **First connection** you may see a prompt like *“Are you sure you want to continue connecting (yes/no)?”* — type **`yes`** and Enter. Then you should see a welcome message and a prompt ending in **`ubuntu@…:~$`**.

5. **If it fails**: wrong IP, security group blocking SSH, wrong key pair for that instance, or wrong username (must be `ubuntu` for Ubuntu AMI).

### 6) Install Docker and deploy (same as Lightsail)

**Where to run this — simple version**

| Steps | Where |
|--------|--------|
| **Steps 1–4** | **Web browser** on your computer → [AWS Console](https://console.aws.amazon.com/ec2/) → EC2 (create instance, security group, copy public IP). No Terminal yet. |
| **Step 7** (`curl` … `/health`) | **Mac Terminal** — run this on your laptop to test the API. |

**Step 5 vs Step 6 (why people get confused)**

- EC2 has **no screen** on your desk. You never “open Terminal *on* EC2” as a separate app. You only use **Terminal on your Mac**.
- **Step 5** — On your Mac, in Terminal: `chmod` then `ssh` (full meaning above under **What Step 5 means**). That **opens** the encrypted login to EC2.
- **Step 6** — You **do not** close Terminal. After `ssh` succeeds, the **same window** shows Ubuntu messages and a prompt like `ubuntu@ip-172-31-…:~$`. Every command you paste next (`sudo apt`, `git clone`, `docker` …) is executed **inside your cloud server**, not on your Mac’s disk. The Mac is only the keyboard and screen; the work happens **on the EC2 instance**.
- When you want your Mac-only prompt back, type `exit` (that ends SSH). Then use a **new** Terminal tab/window on the Mac for Step 7 `curl`.

**Checklist before pasting Step 6 commands:** you already ran `ssh … ubuntu@…`, you see `ubuntu@…:~$`, and you have **not** typed `exit` yet.

On the EC2 instance (same Terminal window as Step 5, **after** `ssh` connected):

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

### 9) Updates after `git push` (manual)

SSH in, pull, rebuild, restart the API container (Postgres can keep running).

**Amazon Linux** (`ec2-user`):

```bash
ssh -i ~/Downloads/your-key.pem ec2-user@YOUR_PUBLIC_IP
cd /opt/marrvaron
git pull origin main
sudo docker build -t marrvaron-api .
NET=$(sudo docker inspect marvaron_postgres -f '{{range $k,$v := .NetworkSettings.Networks}}{{$k}}{{end}}')
sudo docker rm -f marrvaron-api
sudo docker run -d --name marrvaron-api --restart unless-stopped --network "$NET" \
  -e ENVIRONMENT=production -e DB_HOST=postgres -e DB_PORT=5432 \
  -e DB_USER=marvaron_user -e DB_PASSWORD=marvaron_password -e DB_NAME=marvaron_db \
  -e DB_SSLMODE=disable \
  -e JWT_SECRET=use-the-same-secret-as-before \
  -e 'QR_ENCRYPTION_KEY=exactly-32-characters-long-key!!' \
  -e QR_SIGNATURE_SECRET=use-the-same-secret-as-before \
  -p 8080:8080 marrvaron-api
```

Use the **same** `JWT_SECRET` and QR secrets as the first deploy, or existing tokens can break. Then: `curl -s http://127.0.0.1:8080/health`.

**Ubuntu** on EC2: same commands; SSH as `ubuntu` and use your `.pem` path.

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
