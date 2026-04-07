# HTTPS proxy for 4 repositories

Этот bootstrap поднимает Caddy с автоматическими TLS-сертификатами (Let's Encrypt) и проксирует 4 домена на локальные порты сервисов.

## Требования

- DNS A-записи доменов должны указывать на сервер.
- На сервере должен быть установлен Docker + Docker Compose plugin.

## Запуск на сервере

Ниже путь для дефолтного `DEPLOY_PATH` backend-репозитория.

```bash
cd /opt/fizon/backend/ops/https-proxy
chmod +x bootstrap.sh

FRONTEND_DOMAIN=app.example.com \
BACKEND_DOMAIN=api.example.com \
TRACKING_DOMAIN=tracking.example.com \
RECOGNIZING_DOMAIN=recognition.example.com \
LETSENCRYPT_EMAIL=ops@example.com \
./bootstrap.sh
```

## Порты по умолчанию

- `app` -> `127.0.0.1:18081`
- `api` -> `127.0.0.1:18080`
- `tracking` -> `127.0.0.1:18082`
- `recognition` -> `127.0.0.1:18083`

Их можно переопределить через переменные:
`FRONTEND_UPSTREAM_PORT`, `BACKEND_UPSTREAM_PORT`, `TRACKING_UPSTREAM_PORT`, `RECOGNIZING_UPSTREAM_PORT`.
