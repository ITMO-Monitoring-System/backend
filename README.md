# backend

Backend of monitoring system.

## Deploy (push-based)

Push в ветку `main` запускает workflow `.github/workflows/deploy.yml`.

### Required GitHub secrets

- `DEPLOY_HOST` (или `SSH_HOST`)
- `DEPLOY_PORT` (или `SSH_PORT`, по умолчанию `22`)
- `DEPLOY_USERNAME` (или `SSH_USERNAME`)
- `DEPLOY_SSH_PRIVATE_KEY` (или `SSH_PRIVATE_KEY`)
- `DEPLOY_PATH` (опционально, по умолчанию `/opt/fizon/backend`)
- `APP_CONFIG_TOML` (обязательно, содержимое `config.toml` для прода)
- `APP_ENV_FILE` (опционально, содержимое `.env.production`)

### Production files

- `docker-compose.prod.yml`
- `.env.production.example`
- `config.production.example.toml`

## HTTPS proxy for all 4 repositories

Для HTTPS (Let's Encrypt + reverse proxy) используйте:

- `ops/https-proxy/bootstrap.sh`
- `ops/https-proxy/README.md`
