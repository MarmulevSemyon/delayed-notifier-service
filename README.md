# Delayed Notifier Service

Сервис отложенных уведомлений на Go.

Позволяет создать уведомление с заданным временем отправки, сохранить его состояние, поставить задачу в очередь и автоматически выполнить отправку в нужный момент.

Для хранения данных используется **PostgreSQL**, для очередей и отложенной доставки — **RabbitMQ**, для кэширования — **Redis**. Поддерживается отправка уведомлений через Telegram, а также mock-канал для тестирования.

## Возможности

* создание отложенных уведомлений;
* получение уведомления по ID;
* получение списка уведомлений;
* отмена ожидающего уведомления;
* отправка уведомлений в заданное время;
* поддержка нескольких каналов отправки;
* Telegram-уведомления;
* mock-отправитель для тестирования;
* хранение состояния уведомлений в PostgreSQL;
* кэширование уведомлений в Redis;
* асинхронная обработка через RabbitMQ;
* повторная отправка при ошибке;
* экспоненциальная задержка между повторными попытками;
* ограничение максимального количества попыток;
* Docker Compose для запуска всей инфраструктуры;
* HTTP health-check;
* простой web-интерфейс.

## Архитектура

```text
                         HTTP API
                            │
                            ▼
                      ┌───────────┐
                      │  Handler  │
                      └─────┬─────┘
                            │
                            ▼
                      ┌───────────┐
                      │  Service  │
                      └─────┬─────┘
                            │
              ┌─────────────┼─────────────┐
              │             │             │
              ▼             ▼             ▼
        ┌──────────┐  ┌──────────┐  ┌───────────┐
        │PostgreSQL│  │  Redis   │  │ RabbitMQ  │
        └──────────┘  └──────────┘  └─────┬─────┘
                                          │
                                   delayed message
                                          │
                                          ▼
                                     ┌──────────┐
                                     │ Consumer │
                                     └────┬─────┘
                                          │
                                          ▼
                                      ┌────────┐
                                      │ Router │
                                      └───┬────┘
                                  ┌───────┴───────┐
                                  ▼               ▼
                               Mock           Telegram
```

### HTTP API

Принимает запросы пользователя и передаёт их в сервисный слой.

### Service

Содержит основную бизнес-логику:

* проверку входных данных;
* создание уведомления;
* сохранение в PostgreSQL;
* запись в Redis;
* расчёт задержки до отправки;
* публикацию задачи в RabbitMQ;
* получение и отмену уведомлений.

### PostgreSQL

Используется как основное постоянное хранилище уведомлений и их состояния.

### Redis

Используется для кэширования уведомлений.

При запросе уведомления сервис сначала пытается получить его из Redis. Если записи в кэше нет, данные загружаются из PostgreSQL и сохраняются в Redis.

### RabbitMQ

Используется для асинхронной и отложенной обработки уведомлений.

В проекте используются две очереди:

```text
notifications.delay
notifications.ready
```

При создании уведомление помещается в delay-очередь с TTL, соответствующим времени до отправки.

После истечения TTL RabbitMQ перенаправляет сообщение через Dead Letter Exchange в очередь `notifications.ready`, откуда его получает consumer.

Такой подход позволяет реализовать отложенную доставку без постоянного опроса базы данных.

## Жизненный цикл уведомления

После создания уведомление получает статус:

```text
pending
```

Когда consumer начинает обработку:

```text
processing
```

После успешной отправки:

```text
sent
```

При окончательной ошибке:

```text
failed
```

При ручной отмене:

```text
canceled
```

Полный набор состояний:

```text
pending
processing
sent
failed
canceled
```

## Повторные попытки отправки

Если отправить уведомление не удалось, сервис автоматически выполняет повторную попытку.

Используется экспоненциальная задержка:

```text
10 сек.
20 сек.
40 сек.
80 сек.
...
```

После каждой ошибки увеличивается счётчик попыток.

Если количество попыток достигает `max_attempts`, уведомление получает статус:

```text
failed
```

Если `max_attempts` не указан, используется значение:

```text
5
```

Последняя ошибка сохраняется вместе с уведомлением.

## Каналы отправки

В проекте реализованы два канала.

### Mock

```json
{
  "channel": "mock"
}
```

Используется для тестирования логики без фактической отправки сообщения.

### Telegram

```json
{
  "channel": "telegram"
}
```

В качестве `recipient` необходимо передать Telegram `chat_id`.

Для работы Telegram-отправителя требуется токен бота:

```env
TG_BOT_TOKEN=your_bot_token
```

## REST API

| Метод    | Endpoint         | Описание                    |
| -------- | ---------------- | --------------------------- |
| `GET`    | `/health`        | проверка состояния сервиса  |
| `POST`   | `/notify`        | создать уведомление         |
| `GET`    | `/notify/:id`    | получить уведомление по ID  |
| `DELETE` | `/notify/:id`    | отменить уведомление        |
| `GET`    | `/notifications` | получить список уведомлений |
| `GET`    | `/`              | web-интерфейс               |

## Создание уведомления

```http
POST /notify
Content-Type: application/json
```

Пример:

```json
{
  "channel": "telegram",
  "recipient": "123456789",
  "message": "Напоминание о встрече",
  "send_at": "2099-01-01T12:00:00Z",
  "max_attempts": 5
}
```

Поле `send_at` должно быть передано в формате **RFC3339** и содержать время в будущем.

Для тестирования без Telegram можно использовать:

```json
{
  "channel": "mock",
  "recipient": "test-user",
  "message": "Test notification",
  "send_at": "2099-01-01T12:00:00Z",
  "max_attempts": 3
}
```

## Получение уведомления

```http
GET /notify/1
```

Пример ответа:

```json
{
  "id": 1,
  "channel": "telegram",
  "recipient": "123456789",
  "message": "Напоминание о встрече",
  "send_at": "2099-01-01T12:00:00Z",
  "status": "pending",
  "attempts": 0,
  "max_attempts": 5,
  "created_at": "...",
  "updated_at": "..."
}
```

## Отмена уведомления

```http
DELETE /notify/1
```

Ответ:

```json
{
  "status": "canceled"
}
```

## Структура проекта

```text
.
├── cmd
│   └── delayedNotifier
│       └── main.go
│
├── config
│   └── config.go
│
├── internal
│   ├── cache
│   ├── handler
│   ├── model
│   ├── queue
│   ├── repository
│   ├── sender
│   └── service
│
├── migrations
├── static
├── Dockerfile
├── docker-compose.yml
├── .env.ex
├── go.mod
├── go.sum
└── makefile
```

### Основные пакеты

`handler` — HTTP-обработчики.

`service` — бизнес-логика приложения.

`repository` — работа с PostgreSQL.

`cache` — работа с Redis.

`queue` — RabbitMQ producer/consumer и delayed delivery.

`sender` — маршрутизация уведомлений между различными каналами отправки.

`model` — модели данных, статусы и ошибки.

## Запуск через Docker Compose

### 1. Клонировать репозиторий

```bash
git clone https://github.com/MarmulevSemyon/delayed-notifier-service.git
cd delayed-notifier-service
```

### 2. Создать `.env`

В Linux/macOS:

```bash
cp .env.ex .env
```

В PowerShell:

```powershell
Copy-Item .env.ex .env
```

Пример конфигурации:

```env
APP_PORT=8080

POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_DB=notifier
POSTGRES_USER=notifier
POSTGRES_PASSWORD=notifier

RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/

REDIS_ADDR=redis:6379
REDIS_PASSWORD=
REDIS_DB=0

TG_BOT_TOKEN=your_bot_token
```

Для реальной отправки сообщений необходимо указать действующий токен Telegram-бота.

### 3. Запустить приложение

```bash
docker compose up -d --build
```

или:

```bash
make docker-up
```

Docker Compose запускает:

* приложение;
* PostgreSQL;
* Redis;
* RabbitMQ.

Для PostgreSQL, Redis и RabbitMQ настроены health-check'и, а приложение запускается после готовности зависимостей.

### 4. Посмотреть логи

```bash
make docker-logs
```

Логи только приложения:

```bash
make docker-logs-delay
```

### 5. Остановить

```bash
make docker-down
```

## Локальная сборка

```bash
make build
```

Исполняемый файл будет создан в директории:

```text
bin/
```

Проверка проекта:

```bash
go test ./...
```

```bash
go vet ./...
```

Полный набор проверок:

```bash
make all
```

## Использованные технологии

* **Go**
* **PostgreSQL**
* **Redis**
* **RabbitMQ**
* **HTTP / REST API**
* **Telegram Bot API**
* **Docker**
* **Docker Compose**
* **JSON**
* **Makefile**

## Демонстрируемые навыки

### Backend-разработка на Go

Разработка HTTP-сервиса с разделением приложения на transport, service, repository и infrastructure-слои.

### REST API

Реализация HTTP endpoints, обработка JSON, HTTP status codes и валидация входных данных.

### PostgreSQL

Работа с постоянным хранилищем, SQL и миграциями базы данных.

### Redis

Реализация cache-aside подхода: сначала выполняется попытка получить объект из кэша, при отсутствии — чтение из PostgreSQL и последующее заполнение Redis.

### RabbitMQ

Работа с message broker:

* exchange;
* routing keys;
* producer;
* consumer;
* очереди;
* TTL;
* Dead Letter Exchange;
* delayed delivery.

### Асинхронная обработка

Отделение HTTP-запроса создания уведомления от фактической отправки сообщения через очередь сообщений и отдельный consumer.

### Retry-механизм

Реализация повторных попыток отправки с экспоненциальной задержкой и ограничением максимального количества попыток.

### Работа с состоянием

Реализация жизненного цикла уведомления:

```text
pending → processing → sent
                    ↘ failed

pending → canceled
```

### Интеграция с внешним API

Отправка сообщений через Telegram Bot API.

### Dependency separation

Разделение приложения на независимые компоненты:

* handler;
* service;
* repository;
* cache;
* queue;
* sender.

### Docker

Контейнеризация приложения и развёртывание нескольких инфраструктурных компонентов через Docker Compose.

### Конфигурация приложения

Настройка приложения и внешних сервисов через переменные окружения.

## История проекта

Основная цель проекта — практическая работа с backend-архитектурой, брокерами сообщений, кэшированием, базами данных, асинхронной обработкой задач и интеграцией с внешними сервисами.
