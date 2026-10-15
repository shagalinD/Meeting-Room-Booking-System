# Meeting Room Booking System

Бэкенд-сервис для бронирования переговорных комнат. Проект написан на Go и разделён на слои API, бизнес-логики и репозиториев. PostgreSQL хранит данные, Redis используется для кэширования бронирований.

## Возможности

- Регистрация и вход с выдачей access и refresh JWT.
- Создание и просмотр комнат с фильтрами по вместимости.
- Создание бронирования и просмотр бронирований текущего пользователя.
- Хеширование паролей с Argon2id.
- Docker Compose для запуска приложения и зависимостей.
- Nginx как HTTPS-точка входа: HTTP перенаправляется на HTTPS, запросы к Go-серверу проксируются по HTTP внутри сети Compose.
- Swagger UI и OpenAPI-спецификация API.

## Технологии

| Компонент           | Технология                                       |
| ------------------- | ------------------------------------------------ |
| Язык                | Go 1.26.1                                        |
| HTTP-сервер         | `net/http`                                       |
| База данных         | PostgreSQL                                       |
| SQL                 | sqlc                                             |
| Кэш                 | Redis                                            |
| Аутентификация      | JWT                                              |
| Хеширование паролей | Argon2id                                         |
| API-документация    | Swagger (OpenAPI 2.0)                            |
| Веб-сервер и TLS    | Nginx, самоподписанный сертификат для разработки |
| Контейнеризация     | Docker, Docker Compose                           |

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
   ```

   Используйте собственные случайные секреты. Файл `.env` не следует добавлять в Git.

2. Запустите сервисы:

   ```bash
   docker compose up --build
   ```

   Либо используйте `make up`. При первом запуске Compose соберёт Go-сервер и Nginx, а Nginx автоматически создаст самоподписанный сертификат и сохранит его вместе с ключом в Docker volume.

3. Откройте:
   - API и Swagger UI: <https://localhost/swagger/index.html>
   - OpenAPI JSON: <https://localhost/swagger/doc.json>
   - Проверка работоспособности: <https://localhost/health>
   - Adminer: <http://localhost:8083>

   Для локального самоподписанного сертификата браузер покажет предупреждение о недоверенном сертификате. Это ожидаемо для разработки. Для тестового запроса через `curl` можно использовать `-k`; не используйте этот параметр для обычного production-трафика.

   Порт Go-сервера `8081` доступен только внутри Docker-сети. Внешние HTTP-запросы на порт `80` перенаправляются на HTTPS-порт `443`; Nginx проксирует запросы к серверу по внутреннему HTTP.

Остановить сервисы:

```bash
docker compose down
```

Данные PostgreSQL, Redis и TLS-сертификат хранятся в именованных Docker volumes. Для остановки с сохранением данных не добавляйте `-v` к `docker compose down`.

## API

Все API-маршруты и форматы запросов/ответов описаны в Swagger UI. Для защищённых маршрутов передавайте access token в заголовке:

```http
Authorization: Bearer <access_token>
```

| Метод  | Путь                       | Назначение                                                    | Авторизация                  |
| ------ | -------------------------- | ------------------------------------------------------------- | ---------------------------- |
| `GET`  | `/health`                  | Проверка работоспособности сервера                            | Нет                          |
| `POST` | `/auth/register`           | Регистрация пользователя; возвращает пару токенов             | Нет                          |
| `POST` | `/auth/login`              | Вход; возвращает пару токенов                                 | Нет                          |
| `GET`  | `/rooms/list`              | Список комнат; необязательные `min_capacity` и `max_capacity` | Нет                          |
| `POST` | `/rooms/create`            | Создание комнаты                                              | Access token с ролью `admin` |
| `POST` | `/bookings/create`         | Создание бронирования для текущего пользователя               | Access token                 |
| `GET`  | `/bookings/get_by_user_id` | Бронирования текущего пользователя                            | Access token                 |

В текущем роутере отдельный endpoint для обновления токенов не зарегистрирован.

## Миграции и тесты

После запуска PostgreSQL примените миграции:

```bash
make migrate-up
```

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
internal/service/         Бизнес-логика
migrations/              SQL-миграции PostgreSQL
nginx/                   Конфигурация HTTPS reverse proxy и генерация dev-сертификата
pkg/                     Общие пакеты: ошибки, логирование и безопасность
queries/                 SQL-запросы для sqlc
docker-compose.yaml      Оркестрация приложения и зависимостей
Dockerfile               Сборка Go API-сервера
Makefile                 Команды запуска, миграций и тестирования
```
