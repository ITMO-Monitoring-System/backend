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

## Новые API администрирования и отчётов

- `POST /api/user/admin/create`: дополнительные `roles`, `department_id`, `group_code`.
  Создание пользователя, хеша пароля, ролей и связи с группой выполняется в одной
  транзакции. Группа должна принадлежать выбранному направлению; нужна роль `student`.
  Старый формат без ролей и группы остаётся допустимым. Ошибки валидации — 400,
  отсутствующая группа — 404, повторный ИСУ — 409. Доступ только для администратора.
- `PUT /api/departments/{id}`: полный набор `code`, `name`, `alias` (`null` или пустая
  строка очищают псевдоним). Постоянный ID направления сохраняется вместе со всеми
  связями групп. Доступ только для администратора.
- `GET /api/visits/teacher/{subject_id}/report`: отчёт только по собственным начавшимся
  лекциям преподавателя, включая студентов без распознаваний. `date_from` и `date_to`
  принимают RFC3339 или YYYY-MM-DD. Дата конца включается целиком; RFC3339 задаёт
  исключительную верхнюю границу. По умолчанию — последние 90 дней. Максимальный период
  366 дней, `gap_seconds` от 5 до 600 (по умолчанию 180). Более 50 000 строк — ошибка 400,
  без частичного результата. Ответ содержит `lectures` и `items`; `detection_count`
  позволяет отличить отсутствие распознаваний от одиночной отметки с нулевым временем.

Состав групп в отчёте текущий. История переводов между группами и длительность лекции
в схеме отсутствуют, поэтому клиент явно указывает допущения расчёта. Отчёт читает
лекции и посещения в одной транзакции `REPEATABLE READ`. Новые миграции не нужны.

### Проверка

```sh
go test ./...
# Интеграционные тесты создают отдельные временные БД и применяют миграции.
# Нужен тестовый PostgreSQL и право CREATE DATABASE; существующие схемы не изменяются.
TEST_DATABASE_URL='postgres://user@localhost:5432/postgres?sslmode=disable' go test -race ./...
```

Проверяются откат частично созданного пользователя, роли, пароль, выбор группы,
совместимость старого запроса, изменение направления и конфликты кода, границы дат,
права доступа, полнота отчётов, допустимые разрывы и изоляция данных преподавателей.

`go vet ./...` дополнительно сообщает о двух существовавших копированиях
`domain.UserFaces` со встроенным `sync.Mutex` в `dataset_service.go:28` и
`dataset_repository.go:44`. Эти участки не менялись в рамках доработки.
