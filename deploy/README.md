# SocialOps Deployment Files

This directory contains files for deploying SocialOps on Linux servers.

## Deployment Methods

| Method | Best For | Setup Wizard |
|--------|----------|--------------|
| **Docker Compose** | Quick setup, all-in-one | Not needed (auto-setup) |
| **Binary Install** | Production servers, systemd | Web-based wizard |

## Files

| File | Description |
|------|-------------|
| `docker-compose.yml` | Docker Compose configuration (named volumes) |
| `docker-compose.local.yml` | Docker Compose configuration (local directories, easy migration) |
| `docker-deploy.sh` | **One-click Docker deployment script (recommended)** |
| `.env.example` | Docker environment variables template |
| `DOCKER.md` | Docker Hub documentation |
| `install.sh` | One-click binary installation script |
| `socialops.service` | Systemd service unit file |
| `config.example.yaml` | Example configuration file |

---

## Docker Deployment (Recommended)

### Method 1: One-Click Deployment (Recommended)

Use the automated preparation script for the easiest setup:

```bash
# Download and run the preparation script
curl -sSL https://raw.githubusercontent.com/Wei-Shaw/socialops/main/deploy/docker-deploy.sh | bash

# Or download first, then run
curl -sSL https://raw.githubusercontent.com/Wei-Shaw/socialops/main/deploy/docker-deploy.sh -o docker-deploy.sh
chmod +x docker-deploy.sh
./docker-deploy.sh
```

**What the script does:**
- Downloads `docker-compose.local.yml` as `docker-compose.yml` and downloads `.env.example`
- Automatically generates secure secrets (JWT_SECRET, TOTP_ENCRYPTION_KEY, POSTGRES_PASSWORD)
- Creates `.env` file with generated secrets
- Creates necessary data directories (data/, postgres_data/, redis_data/)
- Stores generated secrets in `.env` without printing secret values to the terminal

**After running the script:**
```bash
# Start services
docker compose up -d

# View logs
docker compose logs -f socialops

# If admin password was auto-generated, find it in logs:
docker compose logs socialops | grep "admin password"

# Access Web UI
# http://localhost:8080
```

### Method 2: Manual Deployment

If you prefer manual control:

```bash
# Clone repository
git clone https://github.com/Wei-Shaw/socialops.git
cd socialops/deploy

# Configure environment
cp .env.example .env
nano .env  # Set POSTGRES_PASSWORD and other required variables

# Generate secure secrets (recommended)
JWT_SECRET=$(openssl rand -hex 32)
TOTP_ENCRYPTION_KEY=$(openssl rand -hex 32)
echo "JWT_SECRET=${JWT_SECRET}" >> .env
echo "TOTP_ENCRYPTION_KEY=${TOTP_ENCRYPTION_KEY}" >> .env

# Create data directories
mkdir -p data postgres_data redis_data

# Start all services using local directory version
docker compose -f docker-compose.local.yml up -d

# View logs (check for auto-generated admin password)
docker compose -f docker-compose.local.yml logs -f socialops

# Access Web UI
# http://localhost:8080
```

### Deployment Version Comparison

| Version | Data Storage | Migration | Best For |
|---------|-------------|-----------|----------|
| **docker-compose.local.yml** | Local directories (./data, ./postgres_data, ./redis_data) | ✅ Easy (tar entire directory) | Production, need frequent backups/migration |
| **docker-compose.yml** | Named volumes (/var/lib/docker/volumes/) | ⚠️ Requires docker commands | Simple setup, don't need migration |

**Recommendation:** Use the local-directory compose layout (`docker-compose.local.yml` in the repository, saved as `docker-compose.yml` by `docker-deploy.sh`) for easier data management and migration.

### How Auto-Setup Works

When using Docker Compose with `AUTO_SETUP=true`:

1. On first run, the system automatically:
   - Connects to PostgreSQL and Redis
   - Applies database migrations (SQL files in `backend/migrations/*.sql`) and records them in `schema_migrations`
   - Generates JWT secret (if not provided)
   - Creates admin account (password auto-generated if not provided)
   - Writes config.yaml

2. No manual Setup Wizard needed - just configure `.env` and start

3. If `ADMIN_PASSWORD` is not set, check logs for the generated password:
   ```bash
   docker compose logs socialops | grep "admin password"
   ```

### Database Migration Notes (PostgreSQL)

- Migrations are applied in lexicographic order (e.g. `001_...sql`, `002_...sql`).
- `schema_migrations` tracks applied migrations (filename + checksum).
- Migrations are forward-only; rollback requires a DB backup restore or a manual compensating SQL script.

**Verify `users.allowed_groups` → `user_allowed_groups` backfill**

During the incremental GORM→Ent migration, `users.allowed_groups` (legacy `BIGINT[]`) is being replaced by a normalized join table `user_allowed_groups(user_id, group_id)`.

Run this query to compare the legacy data vs the join table:

```sql
WITH old_pairs AS (
  SELECT DISTINCT u.id AS user_id, x.group_id
  FROM users u
  CROSS JOIN LATERAL unnest(u.allowed_groups) AS x(group_id)
  WHERE u.allowed_groups IS NOT NULL
)
SELECT
  (SELECT COUNT(*) FROM old_pairs)           AS old_pair_count,
  (SELECT COUNT(*) FROM user_allowed_groups) AS new_pair_count;
```

### Commands

For **local directory version** (docker-compose.local.yml):

```bash
# Start services
docker compose -f docker-compose.local.yml up -d

# Stop services
docker compose -f docker-compose.local.yml down

# View logs
docker compose -f docker-compose.local.yml logs -f socialops

# Restart SocialOps only
docker compose -f docker-compose.local.yml restart socialops

# Update to latest version
docker compose -f docker-compose.local.yml pull
docker compose -f docker-compose.local.yml up -d

```

For **named volumes version** (docker-compose.yml):

```bash
# Start services
docker compose up -d

# Stop services
docker compose down

# View logs
docker compose logs -f socialops

# Restart SocialOps only
docker compose restart socialops

# Update to latest version
docker compose pull
docker compose up -d

```

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `POSTGRES_PASSWORD` | **Yes** | - | PostgreSQL password |
| `JWT_SECRET` | **Recommended** | *(auto-generated)* | JWT secret (fixed for persistent sessions) |
| `TOTP_ENCRYPTION_KEY` | **Recommended** | *(auto-generated)* | TOTP encryption key (fixed for persistent 2FA) |
| `SERVER_PORT` | No | `8080` | Server port |
| `ADMIN_EMAIL` | No | `admin@socialops.local` | Admin email |
| `ADMIN_PASSWORD` | No | *(auto-generated)* | Admin password |
| `TZ` | No | `Asia/Shanghai` | Timezone |
| `TWITTER_LOGIN_DEVICE_PARAMS_URL` | No | - | Device fingerprint API for Twitter/X account login |
| `TWITTER_LOGIN_EMAIL_CODE_URL` | No | - | Email verification-code API for Twitter/X account login |
See `.env.example` for all available options.

> **Note:** The `docker-deploy.sh` script automatically generates `JWT_SECRET`, `TOTP_ENCRYPTION_KEY`, and `POSTGRES_PASSWORD` for you.

> **Twitter/X login:** the account workbench `login` action intentionally fails closed with "登录依赖服务未配置，本次未扣费" until the Twitter/X login helper service URLs are configured through `.env` or `config.yaml`.

### Easy Migration (Local Directory Version)

When using `docker-compose.local.yml`, all data is stored in local directories, making migration simple:

```bash
# On source server: Stop services and create archive
cd /path/to/deployment
docker compose -f docker-compose.local.yml down
cd ..
tar czf socialops-complete.tar.gz deployment/

# Transfer to new server
scp socialops-complete.tar.gz user@new-server:/path/to/destination/

# On new server: Extract and start
tar xzf socialops-complete.tar.gz
cd deployment/
docker compose -f docker-compose.local.yml up -d
```

Your entire deployment (configuration + data) is migrated!

## Binary Installation

For production servers using systemd.

### One-Line Installation

```bash
curl -sSL https://raw.githubusercontent.com/Wei-Shaw/socialops/main/deploy/install.sh | sudo bash
```

### Manual Installation

1. Download the latest release from [GitHub Releases](https://github.com/Wei-Shaw/socialops/releases)
2. Extract and copy the binary to `/opt/socialops/`
3. Copy `socialops.service` to `/etc/systemd/system/`
4. Run:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable socialops
   sudo systemctl start socialops
   ```
5. Open the Setup Wizard in your browser to complete configuration

### Commands

```bash
# Install
sudo ./install.sh

# Upgrade
sudo ./install.sh upgrade

# Uninstall
sudo ./install.sh uninstall
```

### Service Management

```bash
# Start the service
sudo systemctl start socialops

# Stop the service
sudo systemctl stop socialops

# Restart the service
sudo systemctl restart socialops

# Check status
sudo systemctl status socialops

# View logs
sudo journalctl -u socialops -f

# Enable auto-start on boot
sudo systemctl enable socialops
```

### Configuration

#### Server Address and Port

During installation, you will be prompted to configure the server listen address and port. These settings are stored in the systemd service file as environment variables.

To change after installation:

1. Edit the systemd service:
   ```bash
   sudo systemctl edit socialops
   ```

2. Add or modify:
   ```ini
   [Service]
   Environment=SERVER_HOST=0.0.0.0
   Environment=SERVER_PORT=3000
   ```

3. Reload and restart:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl restart socialops
   ```

#### Application Configuration

The main config file is at `/etc/socialops/config.yaml` (created by Setup Wizard).

### Prerequisites

- Linux server (Ubuntu 20.04+, Debian 11+, CentOS 8+, etc.)
- PostgreSQL 14+
- Redis 6+
- systemd

### Directory Structure

```
/opt/socialops/
├── socialops              # Main binary
├── socialops.backup       # Backup (after upgrade)
└── data/                # Runtime data

/etc/socialops/
└── config.yaml          # Configuration file
```

---

## Troubleshooting

### Docker

For **local directory version**:

```bash
# Check container status
docker compose -f docker-compose.local.yml ps

# View detailed logs
docker compose -f docker-compose.local.yml logs --tail=100 socialops

# Check database connection
docker compose -f docker-compose.local.yml exec postgres pg_isready

# Check Redis connection
docker compose -f docker-compose.local.yml exec redis redis-cli ping

# Restart all services
docker compose -f docker-compose.local.yml restart

# Check data directories
ls -la data/ postgres_data/ redis_data/
```

For **named volumes version**:

```bash
# Check container status
docker compose ps

# View detailed logs
docker compose logs --tail=100 socialops

# Check database connection
docker compose exec postgres pg_isready

# Check Redis connection
docker compose exec redis redis-cli ping

# Restart all services
docker compose restart
```

### Binary Install

```bash
# Check service status
sudo systemctl status socialops

# View recent logs
sudo journalctl -u socialops -n 50

# Check config file
sudo cat /etc/socialops/config.yaml

# Check PostgreSQL
sudo systemctl status postgresql

# Check Redis
sudo systemctl status redis
```

### Common Issues

1. **Port already in use**: Change `SERVER_PORT` in `.env` or systemd config
2. **Database connection failed**: Check PostgreSQL is running and credentials are correct
3. **Redis connection failed**: Check Redis is running and password is correct
4. **Permission denied**: Ensure proper file ownership for binary install
