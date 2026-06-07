# SocialOps Docker Image

SocialOps is a website account-pool based social operations platform with built-in SaaS foundations for users, subscriptions, payments, and administration.

## Quick Start

```bash
DATABASE_PASSWORD=$(openssl rand -hex 32)

docker run -d \
  --name socialops \
  -p 8080:8080 \
  -e AUTO_SETUP=true \
  -e DATABASE_HOST="host.docker.internal" \
  -e DATABASE_PORT=5432 \
  -e DATABASE_USER="socialops" \
  -e DATABASE_PASSWORD="${DATABASE_PASSWORD}" \
  -e DATABASE_DBNAME="socialops" \
  -e DATABASE_SSLMODE="disable" \
  -e REDIS_HOST="host.docker.internal" \
  -e REDIS_PORT=6379 \
  -e REDIS_PASSWORD="" \
  -e JWT_SECRET="replace-with-32-byte-random-secret" \
  -e TOTP_ENCRYPTION_KEY="replace-with-32-byte-random-secret" \
  -v socialops_data:/app/data \
  weishaw/socialops:latest
```

## Docker Compose

```yaml
services:
  socialops:
    image: weishaw/socialops:latest
    ports:
      - "8080:8080"
    environment:
      - AUTO_SETUP=true
      - SERVER_HOST=0.0.0.0
      - SERVER_PORT=8080
      - SERVER_MODE=release
      - DATABASE_HOST=db
      - DATABASE_PORT=5432
      - DATABASE_USER=postgres
      - DATABASE_PASSWORD=${DATABASE_PASSWORD}
      - DATABASE_DBNAME=socialops
      - DATABASE_SSLMODE=disable
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - JWT_SECRET=replace-with-32-byte-random-secret
      - TOTP_ENCRYPTION_KEY=replace-with-32-byte-random-secret
    volumes:
      - socialops_data:/app/data
    depends_on:
      - db
      - redis

  db:
    image: postgres:18-alpine
    environment:
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=${DATABASE_PASSWORD}
      - POSTGRES_DB=socialops
      - PGDATA=/var/lib/postgresql/data
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:8-alpine
    volumes:
      - redis_data:/data

volumes:
  socialops_data:
  postgres_data:
  redis_data:
```

## Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `DATABASE_HOST` | PostgreSQL host | Yes | - |
| `DATABASE_PORT` | PostgreSQL port | No | `5432` |
| `DATABASE_USER` | PostgreSQL user | No | `socialops` |
| `DATABASE_PASSWORD` | PostgreSQL password | Yes | - |
| `DATABASE_DBNAME` | PostgreSQL database name | No | `socialops` |
| `REDIS_HOST` | Redis host | Yes | - |
| `REDIS_PORT` | Redis port | No | `6379` |
| `REDIS_PASSWORD` | Redis password | No | empty |
| `SERVER_PORT` | Server port inside the container | No | `8080` |
| `SERVER_MODE` | Gin framework mode (`debug`/`release`) | No | `release` |
| `JWT_SECRET` | Fixed JWT signing secret | Recommended | generated during setup |
| `TOTP_ENCRYPTION_KEY` | Fixed TOTP/payment encryption key | Recommended | generated during startup |

## Supported Architectures

- `linux/amd64`
- `linux/arm64`

## Tags

- `latest` - Latest stable release
- `x.y.z` - Specific version
- `x.y` - Latest patch of minor version
- `x` - Latest minor of major version

## Links

- [GitHub Repository](https://github.com/weishaw/socialops)
- [Documentation](https://github.com/weishaw/socialops#readme)
