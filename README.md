# Green-API proxy (Go)

Небольшой HTTP-сервис: браузер и клиенты ходят только в этот backend; он валидирует вход, вызывает официальный [Green-API](https://green-api.com) и возвращает единый JSON-оболочкой. Есть веб-интерфейс для ручных вызовов и Swagger UI с описанием API.

**Базовый URL (после деплоя):** `https://YOUR-SERVICE.onrender.com`  
**Swagger UI:** `https://YOUR-SERVICE.onrender.com/swagger/index.html`  
**Метрики (Prometheus):** `GET /metrics`

## Возможности

- Прокси четырёх методов Green-API: `getSettings`, `getStateInstance`, `sendMessage`, `sendFileByUrl`.
- Версионированный префикс **`/api/v1`** (старые пути **`/api/...`** оставлены как совместимые алиасы).
- Валидация `idInstance`, токена, `chatId`, текста и URL файла на границе сервиса.
- Учётные данные инстанса через заголовки `X-Instance-Id` / `X-Api-Token` (и опционально в теле POST — заголовки перекрывают JSON).
- Единый формат успеха/ошибки; успех включает pretty-print ответа апстрима в поле `pretty`.
- Ограниченные повторы к апстриму при временных сбоях (см. ниже).
- Liveness / readiness: **`/livez`**, **`/readyz`**; **`/healthz`** = тот же ответ, что и liveness.
- Swagger 2.0 из аннотаций ([swaggo/swag](https://github.com/swaggo/swag)) и UI на маршруте `/swagger/`.
- Метрики Prometheus на `/metrics`.

## Архитектура

| Пакет | Роль |
|--------|------|
| `cmd/app` | Точка входа: конфиг, регистрация метрик, клиент Green-API, HTTP-сервер с таймаутами, graceful shutdown. |
| `internal/config` | Переменные окружения (`caarlos0/env`), валидация порта и таймаута. |
| `internal/httpserver` | Chi: request ID, Prometheus HTTP middleware, лог, recover, маршруты, Swagger UI, `/metrics`. |
| `internal/metrics` | Счётчики и гистограммы Prometheus (HTTP и апстрим). |
| `internal/handler` | HTTP-handlers: декод JSON (лимит тела), валидация, вызов прокси, ответы. |
| `internal/greenapi` | HTTP-клиент к Green-API: base URL, таймаут, bounded retry, маппинг ошибок. |
| `internal/domain` | DTO запросов/ответов API и правила валидации. |
| `internal/httpx` | Строгий JSON, заголовки подключения, единый JSON для ошибок. |
| `internal/jsonfmt` | Форматирование сырого JSON апстрима для поля `pretty`. |
| `docs/` | Спецификация OpenAPI (swag): `swagger.json`, `docs.go` (обновление — см. ниже). |
| `web/` | Шаблон `index.html`, стили и скрипты веб-интерфейса. |

### Поток запроса

**Клиент → Chi (middleware: RequestID → Prometheus → лог → recover) → handler → `greenapi.Client` (retry + transport с метриками) → Green-API.**  
Тело ответа апстрима не парсится в структуры — сохраняется как `[]byte` и отдаётся обёрнутым в `pretty`.

## Требования

- Go **1.24+** (в тестах используется `testing.Chdir`; Docker-образ на базе `golang:1.24-alpine`).
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

3. Откройте `http://localhost:8080` (или порт из `APP_PORT`). Swagger: `http://localhost:8080/swagger/index.html`. Метрики: `http://localhost:8080/metrics`.

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
| `GREEN_API_TIMEOUT` | нет | `15s` | Общий дедлайн **одного вызова** клиента к Green-API (включая повторы; см. retry). |

## HTTP API

Общие заголовки для вызовов Green-API:

- `X-Instance-Id` — `idInstance` (4–32 десятичные цифры).
- `X-Api-Token` — `apiTokenInstance` (непустая строка, до 128 символов).

Для `POST` тело — JSON; поля `idInstance` / `apiTokenInstance` в теле допустимы, но **заголовки при совпадении имён перекрывают JSON**.

### Таблица эндпоинтов

| Метод | Путь | Заголовки | Тело | Успешный ответ |
|-------|------|-----------|------|----------------|
| GET | `/livez` | — | — | `200`, текст `ok` |
| GET | `/readyz` | — | — | `200`, текст `ok` |
| GET | `/healthz` | — | — | то же, что `/livez` |
| GET | `/metrics` | — | — | Prometheus text exposition |
| GET | `/` | — | — | `200`, HTML веб-интерфейса |
| GET | `/api/v1/get-settings` | Обязательны `X-Instance-Id`, `X-Api-Token` | — | JSON оболочка, `pretty` — отформатированный ответ Green-API |
| GET | `/api/v1/get-state-instance` | То же | — | То же |
| POST | `/api/v1/send-message` | Рекомендуется то же (или учётные данные в теле) | `Content-Type: application/json` | То же |
| POST | `/api/v1/send-file-by-url` | То же | JSON с `chatId`, `fileUrl`, `fileName`; опционально `caption` | То же |

Эквивалентные пути без `v1`: `/api/get-settings` и т.д. (legacy).

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

Ошибка (клиент/валидация или прокси) — единый объект `error` с **`request_id`** (из middleware Chi, тот же id что в логах):

```json
{
  "ok": false,
  "error": {
    "code": "validation_error",
    "message": "Validation error",
    "request_id": "…",
    "details": { "field": "reason" }
  }
}
```

В ответы **не попадают** сырые тексты апстрима или внутренние стектрейсы; для интеграционных ошибок по-прежнему используется маппинг в `internal/greenapi` (коды вроде `upstream_timeout`, `upstream_rate_limited`, …). В `details` при необходимости остаются обезличенные подсказки (`status`, `retryable`, усечённый `body_snippet`, при 429 — `retry_after`).

Превышение лимита тела JSON (`1 MiB`): `413`, код `payload_too_large`.

### Ограничения домена

- `chatId`: суффикс `@c.us` (личные) или `@g.us` (группы).
- Лимиты и состояние инстанса задаются стороной Green-API.

## Таймауты и HTTP-сервер

| Параметр | Значение | Зачем |
|----------|-----------|--------|
| `ReadHeaderTimeout` | 5s | Защита от slowloris на фазе заголовков. |
| `ReadTimeout` | 30s | Верхняя граница чтения запроса целиком. |
| `WriteTimeout` | 45s | Верхняя граница записи ответа. |
| `IdleTimeout` | 120s | Закрытие неактивных keep-alive соединений. |
| `MaxHeaderBytes` | 1 MiB | Потолок размера заголовков. |
| Лимит JSON-тела (POST) | 1 MiB | `http.MaxBytesReader` в `httpx.DecodeStrictJSON`. |
| `GREEN_API_TIMEOUT` | 15s (env) | Дедлайн контекста на один вызов `greenapi.Client` (все попытки retry внутри него). |

## Повторные запросы к Green-API

Клиент делает до **4 попыток** (первая + до **3** повторов) только если:

- транспортная ошибка (кроме отмены контекста пользователем), или
- HTTP **408**, **429**, **5xx** (502, 503, 504 и т.д. по таблице `HTTPError.Retryable()`),

и при этом **не истёк** контекст/deadline вызова. Повторы **не** выполняются для типичных **4xx** (кроме перечисленных). Между попытками — **экспоненциальная задержка** (от 50 ms с потолком 2 s) и **jitter**. Отмена клиентом (`context.Canceled`) обрабатывается сразу, без «лишних» sleep.

## Метрики (Prometheus)

Эндпоинт: **`GET /metrics`**.

| Метрика | Назначение |
|---------|------------|
| `greenapi_facade_http_requests_total` | Счётчик HTTP-запросов (`method`, `route` — шаблон Chi, `status_class`: 2xx/3xx/4xx/5xx). |
| `greenapi_facade_http_request_duration_seconds` | Длительность обработки HTTP-запроса. |
| `greenapi_facade_upstream_requests_total` | Исходящие запросы к Green-API (**каждая попытка**, включая retry), лейбл `op`. |
| `greenapi_facade_upstream_request_duration_seconds` | Длительность одного исходящего round-trip. |
| `greenapi_facade_upstream_errors_total` | Ошибки апстрима по попытке: `transport`, `canceled`, `rate_limit` (429), `http_5xx`. |

Лейблы держатся **низкозначностными** (без сырого URL, chatId, токенов).

## CI

В репозитории есть GitHub Actions (`.github/workflows/ci.yml`): `go test ./...`, `go vet ./...`, `golangci-lint run`. Конфиг линтера: `.golangci.yml`.

## Примеры curl

```bash
curl -sS http://localhost:8080/api/v1/get-settings \
  -H 'X-Instance-Id: 1101234567' \
  -H 'X-Api-Token: YOUR_TOKEN'
```

```bash
curl -sS http://localhost:8080/api/v1/send-message \
  -H 'Content-Type: application/json; charset=utf-8' \
  -H 'X-Instance-Id: 1101234567' \
  -H 'X-Api-Token: YOUR_TOKEN' \
  -d '{"chatId":"79990000000@c.us","message":"hi"}'
```

Полная машиночитаемая схема: **Swagger UI** (`/swagger/index.html`) и файлы `docs/swagger.json`, `docs/swagger.yaml`.

## Design decisions / trade-offs

- **Сырой JSON от Green-API.** Клиент прокси возвращает `[]byte`, ответ отдаётся в `pretty` без привязки к конкретной версии схемы Green-API. Плюс: меньше ломается при изменениях апстрима; минус: типобезопасность только на границе своего API.
- **Небольшой bounded retry.** Повторы ограничены, с backoff и уважением к `context`; снижают шум от кратковременных 503/сети, но **POST** к апстриму теоретически могут выполниться более одного раза при сбое после принятия запроса апстримом — осознанный компромисс для этого маленького фасада.
- **Учётные данные в заголовках на своём boundary.** Так проще вызывать API из браузера и из `curl`, не светя токен в URL; заголовки перекрывают JSON.
- **Swagger 2.0 через аннотации.** Спецификация живёт рядом с кодом; отдельный контур «схема → клиент» здесь не используется.
- **Намеренно компактный сервис:** нет отдельного слоя repository, очередей и т.д. — только то, что повышает надёжность и наблюдаемость без раздувания архитектуры.

## Возможные улучшения

- Расширить `/readyz` реальной проверкой зависимостей (БД, очередь), когда они появятся.
- OpenAPI 3 / отдельный redoc при росте контракта.
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
internal/metrics
internal/handler
internal/greenapi
internal/domain
internal/httpx
internal/jsonfmt
docs/                 # OpenAPI (swag)
web/templates/index.html
web/static/
.github/workflows/
Dockerfile
docker-compose.yml
Taskfile.yml
```
