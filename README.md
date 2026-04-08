# Green-API proxy (Go)

Небольшой HTTP-сервис: браузер и клиенты ходят только в этот backend; он валидирует вход, вызывает официальный [Green-API](https://green-api.com) и возвращает единый JSON-оболочкой. Есть веб-интерфейс для ручных вызовов и Swagger UI с описанием API.

**Базовый URL (после деплоя):** `https://YOUR-SERVICE.onrender.com`
**Swagger UI:** `https://YOUR-SERVICE.onrender.com/swagger/index.html`

## Возможности

- Прокси четырёх методов Green-API: `getSettings`, `getStateInstance`, `sendMessage`, `sendFileByUrl`.
- Валидация `idInstance`, токена, `chatId`, текста и URL файла на границе сервиса.
- Учётные данные инстанса через заголовки `X-Instance-Id` / `X-Api-Token` (и опционально в теле POST — заголовки перекрывают JSON).
- Единый формат успеха/ошибки; успех включает pretty-print ответа апстрима в поле `pretty`.
- `GET /healthz`, статика и HTML-форма для быстрой проверки методов.
- Swagger 2.0 из аннотаций ([swaggo/swag](https://github.com/swaggo/swag)) и UI на маршруте `/swagger/`.

## Архитектура

| Пакет | Роль |
|--------|------|
| `cmd/app` | Точка входа: конфиг, клиент Green-API, HTTP-сервер, graceful shutdown. |
| `internal/config` | Переменные окружения (`caarlos0/env`), валидация порта и таймаута. |
| `internal/httpserver` | Chi: middleware (request ID, лог, recover), маршруты, Swagger UI. |
| `internal/handler` | HTTP-handlers: декод JSON, валидация, вызов прокси, ответы. |
| `internal/greenapi` | HTTP-клиент к Green-API: base URL, таймаут, маппинг ошибок. |
| `internal/domain` | DTO запросов/ответов API и правила валидации. |
| `internal/httpx` | Строгий JSON, заголовки подключения, запись JSON-ответов. |
| `internal/jsonfmt` | Форматирование сырого JSON апстрима для поля `pretty`. |
| `docs/` | Спецификация OpenAPI (swag): `swagger.json`, `docs.go` (обновление — см. ниже). |
| `web/` | Шаблон `index.html`, стили и скрипты веб-интерфейса. |

Поток: **клиент → handler → greenapi.Client → Green-API**; тело ответа апстрима не парсится в структуры — сохраняется как `[]byte` и отдаётся обёрнутым.

## Требования

- Go **1.22+**
- [Task](https://taskfile.dev/installation/) (опционально, для алиасов из `Taskfile.yml`)

## Локальный запуск

1. При необходимости скопируйте переменные окружения (при старте подхватывается `.env` через `godotenv`):

   ```bash
   cp .env.example .env
   ```

2. Запуск:

   ```bash
   task run
   ```

   или

   ```bash
   go run ./cmd/app
   ```

3. Откройте `http://localhost:8080` (или порт из `APP_PORT`). Swagger: `http://localhost:8080/swagger/index.html`.

### Обновление спецификации Swagger

Нужен CLI той же major-линии, что и `github.com/swaggo/swag` в `go.mod` (сейчас **v1.16.x**):

```bash
go install github.com/swaggo/swag/cmd/swag@v1.16.4
task swagger
```

Либо напрямую: `swag init -g cmd/app/main.go -o docs --parseDependency --parseInternal`

## Docker

Образ: multi-stage сборка, финальный слой `distroless`, в образ копируется бинарник и каталог `web/`. Рабочая директория в контейнере совпадает с layout образа, `WEB_ROOT` по умолчанию не обязателен.

```bash
cp .env.example .env   # опционально
task docker:up
```

Остановка: `task docker:down`.

Без Task:

```bash
docker compose up --build
```

Сервис слушает `APP_HOST`:`APP_PORT` (см. таблицу ниже).

## Переменные окружения

| Переменная | Обязательная | По умолчанию | Описание |
|------------|--------------|--------------|----------|
| `APP_HOST` | нет | `0.0.0.0` | Адрес привязки HTTP-сервера. |
| `APP_PORT` | нет | `8080` | Порт (1–65535). Если не задан, читается стандартный **`PORT`** (удобно для Render и аналогов). |
| `PORT` | нет | — | Используется только когда **`APP_PORT` не задан**; типично выставляется платформой деплоя. |
| `WEB_ROOT` | нет | `.` | Корень проекта для `web/templates/index.html` и статики; при старте проверяется наличие шаблона. |
| `GREEN_API_BASE_URL` | нет | `https://api.green-api.com` | Базовый URL Green-API. |
| `GREEN_API_TIMEOUT` | нет | `15s` | Таймаут HTTP-клиента к Green-API. |

## HTTP API

Общие заголовки для вызовов Green-API:

- `X-Instance-Id` — `idInstance` (4–32 десятичные цифры).
- `X-Api-Token` — `apiTokenInstance` (непустая строка, до 128 символов).

Для `POST` тело — JSON; поля `idInstance` / `apiTokenInstance` в теле допустимы, но **заголовки при совпадении имён перекрывают JSON**.

### Таблица эндпоинтов

| Метод | Путь | Заголовки | Тело | Успешный ответ |
|-------|------|-----------|------|----------------|
| GET | `/healthz` | — | — | `200`, текст `ok` |
| GET | `/` | — | — | `200`, HTML веб-интерфейса |
| GET | `/api/get-settings` | Обязательны `X-Instance-Id`, `X-Api-Token` | — | JSON оболочка, `pretty` — отформатированный ответ Green-API |
| GET | `/api/get-state-instance` | То же | — | То же |
| POST | `/api/send-message` | Рекомендуется то же (или учётные данные в теле) | `Content-Type: application/json` | То же |
| POST | `/api/send-file-by-url` | То же | JSON с `chatId`, `fileUrl`, `fileName`; опционально `caption` | То же |

Статика: `GET /static/*`.

### Примеры тел POST

`send-message`:

```json
{
  "idInstance": "1101234567",
  "apiTokenInstance": "your-token",
  "chatId": "79990000000@c.us",
  "message": "Hello"
}
```

`send-file-by-url`:

```json
{
  "chatId": "79990000000@c.us",
  "fileUrl": "https://example.com/file.jpg",
  "fileName": "file.jpg",
  "caption": "optional"
}
```

### Формат ответа

Успех (`200`):

```json
{
  "ok": true,
  "pretty": "{\n  \"key\": \"value\"\n}"
}
```

Ошибка (клиент/валидация или прокси):

```json
{
  "ok": false,
  "error": {
    "code": "validation_error",
    "message": "Validation error",
    "details": { "field": "reason" }
  }
}
```

Типичные коды ошибок прокси: `upstream_timeout`, `upstream_rate_limited`, `upstream_auth_error`, `upstream_invalid_response`; в `details` могут быть `status`, `retryable`, `body_snippet`, при 429 — `retry_after`.

### Ограничения домена

- `chatId`: суффикс `@c.us` (личные) или `@g.us` (группы).
- Лимиты и состояние инстанса задаются стороной Green-API.

## Примеры curl

```bash
curl -sS http://localhost:8080/api/get-settings \
  -H 'X-Instance-Id: 1101234567' \
  -H 'X-Api-Token: YOUR_TOKEN'
```

```bash
curl -sS http://localhost:8080/api/send-message \
  -H 'Content-Type: application/json; charset=utf-8' \
  -H 'X-Instance-Id: 1101234567' \
  -H 'X-Api-Token: YOUR_TOKEN' \
  -d '{"chatId":"79990000000@c.us","message":"hi"}'
```

Полная машиночитаемая схема: **Swagger UI** (`/swagger/index.html`) и файлы `docs/swagger.json`, `docs/swagger.yaml`.

## Design decisions / trade-offs

- **Сырой JSON от Green-API.** Клиент прокси возвращает `[]byte`, ответ отдаётся в `pretty` без привязки к конкретной версии схемы Green-API. Плюс: меньше ломается при изменениях апстрима; минус: типобезопасность только на границе своего API, не внутри полей Green-API.
- **Без retry.** Повторы с токенами и неидемпотентными POST усложняют семантику и могут усугубить rate limit. Клиент или оркестратор могут решать политику повторов осознанно.
- **Учётные данные в заголовках на своём boundary.** Так проще вызывать API из браузера и из `curl`, не светя токен в URL; заголовки перекрывают JSON, чтобы один и тот же контракт работал и для GET-only сценариев.
- **Swagger 2.0 через аннотации.** Спецификация живёт рядом с кодом и обновляется вместе с handlers; отдельный контур «схема → клиент» здесь не используется.

## Возможные улучшения

- Метрики (Prometheus), трассировка, бюджетные лимиты на размер тел.
- Опциональный retry только для идемпотентных GET с backoff и учётом `Retry-After`.
- Версионирование API (`/v1/...`), отдельный redoc/openapi3 при росте контракта.
- E2E с записанным Green-API или wiremock.

## Деплой

По соотношению «скорость / простота» удобен **[Render](https://render.com) Web Service**: Dockerfile уже есть, HTTPS выдаётся автоматически, достаточно указать запуск контейнера; выданный платформой **`PORT`** подхватывается, если не задан `APP_PORT`. На бесплатном тарифе сервис может засыпать при простое.

**Fly.io** подойдёт, если важен явный Docker-цикл и конфигурация вроде `fly.toml`; стартовых шагов обычно больше, чем у Render.

**Render** — если нужен быстрый публичный URL с минимальной настройкой; **Fly.io** — если приоритет контейнерному деплою и больший контроль.

После выкладки подставьте в начало README фактические URL сервиса и Swagger UI.

## Структура репозитория

```
cmd/app/main.go
internal/config
internal/httpserver
internal/handler
internal/greenapi
internal/domain
internal/httpx
internal/jsonfmt
docs/                 # OpenAPI (swag)
web/templates/index.html
web/static/
Dockerfile
docker-compose.yml
Taskfile.yml
```
