# Meeting Room Booking System

Система бронирования переговорных комнат на Go. HTTP API Gateway направляет запросы в отдельные gRPC-сервисы Auth, Room и Booking. Сервисы используют PostgreSQL, Booking дополнительно использует Redis для кэширования, а события создания бронирований доставляются через Kafka с transactional outbox.

## Возможности

- Регистрация и вход с выдачей access и refresh JWT.
- Создание и просмотр комнат с фильтрами по вместимости.
- Создание бронирования и просмотр бронирований текущего пользователя.
- Публикация событий `booking.created` через Kafka с transactional outbox и повторными попытками.
- Хеширование паролей с Argon2id.
- Docker Compose для запуска приложения и зависимостей.
- Nginx как HTTPS-точка входа: HTTP перенаправляется на HTTPS, запросы к Go-серверу проксируются по HTTP внутри сети Compose.
- API Gateway связывает HTTP-контракты с gRPC-сервисами; `/ready` проверяет готовность всех upstream-сервисов.
- Swagger UI и OpenAPI-спецификация API.

## Технологии

| Компонент           | Технология                                       |
| ------------------- | ------------------------------------------------ |
| Язык                | Go 1.26.1                                        |
| HTTP-сервер         | `net/http`                                       |
| База данных         | PostgreSQL                                       |
| SQL                 | sqlc                                             |
| Кэш                 | Redis                                            |
| События             | Apache Kafka                                     |
| Аутентификация      | JWT                                              |
| Хеширование паролей | Argon2id                                         |
| API-документация    | Swagger (OpenAPI 2.0)                            |
| Веб-сервер и TLS    | Nginx, самоподписанный сертификат для разработки |
| Контейнеризация     | Docker, Docker Compose                           |

## Booking gRPC service

Сервис бронирований запускается отдельным Go-процессом из `cmd/booking`.
Он реализует версионированный контракт `BookingService` из модуля
`github.com/shagalinD/booking-service-proto/gen/go` и слушает
`BOOKING_GRPC_ADDR` (по умолчанию `:50051`). Для вызовов RPC передавайте
access-токен в gRPC metadata: `authorization: Bearer <token>`.

Запустить сервис вместе с PostgreSQL и Redis можно командами
`make run-booking` или `docker compose up --build booking`. В Compose порт
gRPC доступен только внутри Docker-сети; для локального gRPC-клиента при
необходимости добавьте публикацию порта на хост.

HTTP API Gateway передаёт запросы бронирований этому сервису. Gateway пересылает access-токен через gRPC metadata; Booking сам проверяет токен и определяет пользователя.

При создании бронирования сервис атомарно сохраняет запись и событие `booking.created`
в PostgreSQL. Фоновый publisher отправляет события в Kafka topic `booking.events`;
при недоступности брокера событие остаётся в outbox и будет повторено. Доставка
имеет семантику at-least-once, поэтому будущие consumers должны быть идемпотентны.
Событие содержит `id`, `type`, `schema_version`, `occurred_at` и `data` с ID
бронирования, пользователя и комнаты, временем, названием и статусом.

## Room gRPC service

Сервис комнат запускается отдельным Go-процессом из `cmd/room`. Он реализует
версионированный контракт `RoomService`, слушает `ROOM_GRPC_ADDR` (по
умолчанию `:50052`) и использует PostgreSQL. Создание комнаты требует
проверенного access-токена пользователя с ролью `admin`; список и получение
комнат доступны без токена. Передавайте токен в gRPC metadata как
`authorization: Bearer <token>`.

Запустить сервис можно командой `make run-room` или
`docker compose up --build room`. Порт gRPC доступен только внутри Docker-сети.
Перед первым использованием примените миграции БД командой `make migrate-up`;
она добавляет столбец описания комнаты и сохраняет введённые описания.

## Auth gRPC service

Сервис аутентификации запускается отдельным Go-процессом из `cmd/auth`.
Он реализует версионированный контракт `AuthService`, слушает
`AUTH_GRPC_ADDR` (по умолчанию `:50053`) и использует PostgreSQL. RPC
`Register` создаёт обычного пользователя; роль администратора нельзя задать
через регистрацию. `Login` проверяет пароль и выдаёт пару JWT, а
`RefreshTokens` обновляет её по refresh-токену.

Запустить сервис можно командой `make run-auth` или
`docker compose up --build auth`. Порт gRPC доступен только внутри Docker-сети.

## Запуск через Docker Compose

Требуются Docker и Docker Compose.

1. Создайте `.env` в корне проекта. Например:

   ```env
   POSTGRES_USER=postgres
   POSTGRES_PASSWORD=replace-with-a-database-password
   POSTGRES_DB=postgres
   POSTGRES_PORT=5434
   POSTGRES_HOST=localhost

   REDIS_USER=default
   REDIS_PASSWORD=replace-with-a-redis-password
   REDIS_USER_PASSWORD=replace-with-a-redis-password
   REDIS_PORT=6379
   REDIS_HOST=localhost
   REDIS_INDEX=0
   REDIS_TIME_TO_LIVE=5m

   LOG_LEVEL=debug
   JWT_ACCESS_SECRET=replace-with-a-long-random-access-secret
   JWT_REFRESH_SECRET=replace-with-a-long-random-refresh-secret
   SERVER_PORT=:8081

   KAFKA_BROKERS=localhost:9092
   KAFKA_TOPIC=booking.events
   KAFKA_CONSUMER_GROUP=booking-event-logger
   ```

   Используйте собственные случайные секреты. Файл `.env` не следует добавлять в Git.

2. Запустите сервисы:

   ```bash
   docker compose up --build
   ```

   Либо используйте `make up`. При первом запуске Compose соберёт API Gateway, все gRPC-сервисы, Kafka consumer и Nginx, а также запустит брокер Kafka. В Compose адрес брокера задаётся как `kafka:9092`; значения `KAFKA_BROKERS` в `.env` используются при локальном запуске Go-процессов вне Compose. Nginx автоматически создаст самоподписанный сертификат и сохранит его вместе с ключом в Docker volume.

3. Откройте:
   - API и Swagger UI: <https://localhost/swagger/index.html>
   - OpenAPI JSON: <https://localhost/swagger/doc.json>
   - Проверка работоспособности: <https://localhost/health>
   - Готовность API Gateway и всех gRPC-сервисов: <https://localhost/ready>
   - Adminer: <http://localhost:8083>

   Для локального самоподписанного сертификата браузер покажет предупреждение о недоверенном сертификате. Это ожидаемо для разработки. Для тестового запроса через `curl` можно использовать `-k`; не используйте этот параметр для обычного production-трафика.

   Порт Go-сервера `8081` доступен только внутри Docker-сети. Внешние HTTP-запросы на порт `80` перенаправляются на HTTPS-порт `443`; Nginx проксирует запросы к серверу по внутреннему HTTP.

Остановить сервисы:

```bash
docker compose down
```

Данные PostgreSQL, Redis, Kafka и TLS-сертификат хранятся в именованных Docker volumes. Для остановки с сохранением данных не добавляйте `-v` к `docker compose down`.

## API

Все API-маршруты и форматы запросов/ответов описаны в Swagger UI. Для защищённых маршрутов передавайте access token в заголовке:

```http
Authorization: Bearer <access_token>
```

| Метод  | Путь                       | Назначение                                                    | Авторизация                  |
| ------ | -------------------------- | ------------------------------------------------------------- | ---------------------------- |
| `GET`  | `/health`                  | Проверка работоспособности сервера                            | Нет                          |
| `GET`  | `/ready`                   | Готовность API Gateway и всех gRPC-сервисов                   | Нет                          |
| `POST` | `/auth/register`           | Регистрация пользователя; возвращает пару токенов             | Нет                          |
| `POST` | `/auth/login`              | Вход; возвращает пару токенов                                 | Нет                          |
| `POST` | `/auth/refresh`            | Обновление пары токенов по refresh-токену                     | Нет                          |
| `GET`  | `/rooms/list`              | Список комнат; необязательные `min_capacity` и `max_capacity` | Нет                          |
| `GET`  | `/rooms/{id}`              | Получение комнаты по ID                                       | Нет                          |
| `POST` | `/rooms/create`            | Создание комнаты                                              | Access token с ролью `admin` |
| `POST` | `/bookings/create`         | Создание бронирования для текущего пользователя               | Access token                 |
| `GET`  | `/bookings/get_by_user_id` | Бронирования текущего пользователя                            | Access token                 |

## Миграции и тесты

После запуска PostgreSQL примените миграции:

```bash
make migrate-up
```

Миграция `0003_booking_event_outbox` создаёт таблицу transactional outbox для Kafka.
Её необходимо применить до создания бронирований новой версией сервиса.

Откатить последнюю миграцию:

```bash
make migrate-down
```

Запустить unit-тесты без интеграционных проверок PostgreSQL:

```bash
go test -short ./...
```

Полный набор тестов:

```bash
go test ./...
```

Интеграционные тесты репозитория PostgreSQL требуют работающую базу. По умолчанию они используют `localhost:5434` и базу `test_db`; параметры подключения можно задать переменными окружения `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD` и `POSTGRES_DB`.

## Обновление Swagger-документации

Swagger-описания находятся в комментариях к HTTP-обработчикам. После их изменения перегенерируйте файлы в `docs/`:

```bash
go install github.com/swaggo/swag/cmd/swag@v1.16.4
swag init -g cmd/api/main.go -o docs
```

## Структура проекта

```text
cmd/api/                 Точка входа и регистрация HTTP-маршрутов
docs/                    Сгенерированная Swagger/OpenAPI-документация
internal/api/            HTTP-обработчики, middleware и DTO
internal/config/         Загрузка конфигурации и подключение к БД/Redis
internal/db/             Код SQL, сгенерированный sqlc
internal/domain/          Доменные модели и интерфейсы
internal/repository/      Реализации репозиториев PostgreSQL и Redis
internal/messaging/kafka/ Kafka producer, consumer и настройки подключения
internal/outbox/          Публикация событий из transactional outbox
internal/service/         Бизнес-логика
migrations/              SQL-миграции PostgreSQL
cmd/kafka-consumer/       Демонстрационный consumer событий бронирований
nginx/                   Конфигурация HTTPS reverse proxy и генерация dev-сертификата
pkg/                     Общие пакеты: ошибки, логирование и безопасность
queries/                 SQL-запросы для sqlc
docker-compose.yaml      Оркестрация приложения и зависимостей
Dockerfile               Сборка Go API-сервера
Makefile                 Команды запуска, миграций и тестирования
```
