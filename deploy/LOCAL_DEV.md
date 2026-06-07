# Local Host Development

This project is easiest to debug locally with:

- PostgreSQL and Redis in Docker
- Go backend running on the host with auto-restart
- Vue/Vite frontend running on the host with hot module reload

## What gets created

- `deploy/.env.host-dev`
  - Local development environment
  - Admin account is preset to `3081794680@qq.com` / `668435li`
- `deploy/dev-data`
  - Backend `config.yaml` and `.installed`
- `deploy/dev-postgres_data`
  - PostgreSQL data
- `deploy/dev-redis_data`
  - Redis data

## Start dependency services

```bash
make dev-services-up
```

This starts:

- PostgreSQL on `127.0.0.1:5433`
- Redis on `127.0.0.1:6380`

## Start backend with auto-restart

```bash
make dev-backend
```

Notes:

- First run uses `AUTO_SETUP=true`
- Config is written to `deploy/dev-data/config.yaml`
- Admin user is auto-created from `deploy/.env.host-dev`
- Backend listens on `http://127.0.0.1:8080`

## Start frontend with hot update

In a second terminal:

```bash
make dev-frontend
```

Frontend runs on:

- `http://127.0.0.1:3000`

Vite proxies `/api`, `/v1`, and `/setup` to the local backend automatically.

## Stop services

```bash
make dev-services-down
```

## Reset local development data

If you want a completely fresh local environment, stop the containers and remove:

```bash
rm -rf deploy/dev-data deploy/dev-postgres_data deploy/dev-redis_data
```

After that, rerun `make dev-services-up` and `make dev-backend`.
