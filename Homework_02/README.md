# Marketplace API

REST API сервис маркетплейса на Go с контрактным подходом (OpenAPI), JWT-авторизацией и ролевой моделью.

## Технологии

- **Go 1.24** — язык реализации
- **PostgreSQL 16** — база данных
- **oapi-codegen** — кодогенерация из OpenAPI 3.0.3
- **pgx/v5** — драйвер PostgreSQL
- **Flyway** — миграции БД
- **golang-jwt/v5** — JWT-токены
- **Docker** — контейнеризация


## Быстрый старт

### Запуск через Docker Compose

```bash
make up
```

Это поднимет:
- PostgreSQL на порту `55432`
- Flyway (применит миграции)
- Приложение на порту `8080`
- pgAdmin на порту `5050`

### Остановка

```bash
make down
```

### Другие команды

```bash
make logs        # Логи всех сервисов
make ps          # Статус контейнеров
make db-shell    # psql к БД
make generate    # Перегенерация кода из OpenAPI
```

## API-эндпоинты

### Авторизация

| Метод  | Endpoint         | Описание                              | Доступ     |
|--------|------------------|---------------------------------------|------------|
| POST   | /auth/register   | Регистрация пользователя              | Публичный  |
| POST   | /auth/login      | Логин (access + refresh токены)       | Публичный  |
| POST   | /auth/refresh    | Обновление access token               | Публичный  |

### Товары

| Метод  | Endpoint         | Описание                            | Доступ           |
|--------|------------------|-------------------------------------|------------------|
| POST   | /products        | Создать товар                       | SELLER, ADMIN    |
| GET    | /products        | Список товаров (пагинация, фильтры) | Все              |
| GET    | /products/{id}   | Получить товар по ID                | Все              |
| PUT    | /products/{id}   | Обновить товар                      | SELLER (свои), ADMIN |
| DELETE | /products/{id}   | Мягкое удаление (ARCHIVED)          | SELLER (свои), ADMIN |

### Заказы

| Метод  | Endpoint              | Описание                          | Доступ           |
|--------|-----------------------|-----------------------------------|------------------|
| POST   | /orders               | Создать заказ                     | USER, ADMIN      |
| GET    | /orders/{id}          | Получить заказ по ID              | USER (свои), ADMIN |
| PUT    | /orders/{id}          | Обновить заказ                    | USER (свои), ADMIN |
| POST   | /orders/{id}/cancel   | Отменить заказ                    | USER (свои), ADMIN |

### Промокоды

| Метод  | Endpoint       | Описание          | Доступ        |
|--------|----------------|--------------------|---------------|
| POST   | /promo-codes   | Создать промокод   | SELLER, ADMIN |

## Примеры использования

### 1. Регистрация и авторизация

```bash
# Регистрация покупателя
curl -s -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"buyer1","password":"pass123","role":"USER"}' | jq .
# → 201: {"id":"...","username":"buyer1","role":"USER","created_at":"..."}

# Регистрация продавца
curl -s -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"seller1","password":"pass123","role":"SELLER"}' | jq .
# → 201: {"id":"...","username":"seller1","role":"SELLER","created_at":"..."}

# Регистрация администратора
curl -s -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"admin1","password":"pass123","role":"ADMIN"}' | jq .
# → 201: {"id":"...","username":"admin1","role":"ADMIN","created_at":"..."}

# Логин (получаем токены)
curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"seller1","password":"pass123"}' | jq .
# → 200: {"access_token":"eyJ...","refresh_token":"eyJ..."}

# Сохраняем токены в переменные для удобства
SELLER_TOKEN="<access_token продавца из логина>"
BUYER_TOKEN="<access_token покупателя из логина>"

# Обновление access token по refresh token
curl -s -X POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<refresh_token из логина>"}' | jq .
# → 200: {"access_token":"eyJ...","refresh_token":"eyJ..."}

# Повторная регистрация с тем же username
curl -s -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"buyer1","password":"other","role":"USER"}' | jq .
# → 409: {"error_code":"USER_ALREADY_EXISTS","message":"..."}

# Логин с неверным паролем
curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"buyer1","password":"wrong"}' | jq .
# → 401: {"error_code":"INVALID_CREDENTIALS","message":"..."}
```

### 2. Товары (SELLER)

```bash
# Создать товар
curl -s -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -d '{"name":"Laptop","price":1000,"stock":10,"category":"Electronics","status":"ACTIVE"}' | jq .
# → 201: {"id":"<laptop_id>","name":"Laptop","price":1000,"stock":10,...}

curl -s -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -d '{"name":"Mouse","price":50,"stock":5,"category":"Electronics","status":"ACTIVE"}' | jq .
# → 201: {"id":"<mouse_id>","name":"Mouse","price":50,"stock":5,...}

# Обновить товар
curl -s -X PUT http://localhost:8080/products/<laptop_id> \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -d '{"name":"Laptop Pro","price":1200,"stock":10,"category":"Electronics","status":"ACTIVE"}' | jq .
# → 200: {"name":"Laptop Pro","price":1200,...}

# Мягкое удаление (перевод в ARCHIVED)
curl -s -X DELETE http://localhost:8080/products/<laptop_id> \
  -H "Authorization: Bearer $SELLER_TOKEN" | jq .
# → 200: {"status":"ARCHIVED",...}

# Список товаров (публичный, без авторизации)
curl -s http://localhost:8080/products | jq .
# → 200: {"content":[...],"totalElements":2,"page":0,"size":20}

# Пагинация
curl -s "http://localhost:8080/products?page=0&size=1" | jq .
# → 200: {"content":[1 товар],"totalElements":2,"page":0,"size":1}

# Фильтрация по статусу
curl -s "http://localhost:8080/products?status=ACTIVE" | jq .
# → 200: только активные товары

# Фильтрация по категории
curl -s "http://localhost:8080/products?category=Electronics" | jq .
# → 200: только товары из категории Electronics

# Получить товар по ID
curl -s http://localhost:8080/products/<laptop_id> | jq .
# → 200: {"id":"...","name":"Laptop",...}
```

### 3. Валидация

```bash
# Пустое имя товара
curl -s -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -d '{"name":"","price":1000,"stock":10,"category":"Electronics","status":"ACTIVE"}' | jq .
# → 400: {"error_code":"VALIDATION_ERROR","message":"...","details":{"name":"..."}}

# Отрицательная цена
curl -s -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -d '{"name":"Test","price":-5,"stock":10,"category":"Electronics","status":"ACTIVE"}' | jq .
# → 400: {"error_code":"VALIDATION_ERROR","message":"...","details":{"price":"..."}}

# Несуществующий товар
curl -s http://localhost:8080/products/00000000-0000-0000-0000-000000000099 | jq .
# → 404: {"error_code":"PRODUCT_NOT_FOUND","message":"..."}
```

### 4. Промокоды (SELLER/ADMIN)

```bash
# Создание промокода PERCENTAGE 10%
curl -s -X POST http://localhost:8080/promo-codes \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -d '{"code":"SAVE10","discount_type":"PERCENTAGE","discount_value":10,"min_order_amount":100,"max_uses":5,"valid_from":"2025-01-01T00:00:00Z","valid_until":"2027-01-01T00:00:00Z"}' | jq .
# → 201: {"code":"SAVE10","discount_type":"PERCENTAGE","discount_value":10,...}

# Создание промокода FIXED_AMOUNT
curl -s -X POST http://localhost:8080/promo-codes \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -d '{"code":"FLAT500","discount_type":"FIXED_AMOUNT","discount_value":500,"min_order_amount":1000,"max_uses":10,"valid_from":"2025-01-01T00:00:00Z","valid_until":"2027-01-01T00:00:00Z"}' | jq .
# → 201: {"code":"FLAT500","discount_type":"FIXED_AMOUNT","discount_value":500,...}
```

### 5. Заказы (USER)

```bash
# Создание заказа с промокодом
curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d '{"items":[{"product_id":"<laptop_id>","quantity":2},{"product_id":"<mouse_id>","quantity":1}],"promo_code":"SAVE10"}' | jq .
# → 201: {"status":"CREATED","total_amount":1845,"discount_amount":205,"items":[...]}

# Проверка stock уменьшился
curl -s http://localhost:8080/products/<laptop_id> | jq .stock
# → 8 (было 10, заказано 2)

# Получить заказ по ID
curl -s http://localhost:8080/orders/<order_id> \
  -H "Authorization: Bearer $BUYER_TOKEN" | jq .
# → 200: {"id":"...","status":"CREATED","items":[...],...}

# Обновить заказ (изменить позиции)
curl -s -X PUT http://localhost:8080/orders/<order_id> \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d '{"items":[{"product_id":"<mouse_id>","quantity":3}]}' | jq .
# → 200: старый stock восстановлен, новый зарезервирован, total_amount пересчитан

# Отмена заказа
curl -s -X POST http://localhost:8080/orders/<order_id>/cancel \
  -H "Authorization: Bearer $BUYER_TOKEN" | jq .
# → 200: {"status":"CANCELED",...}

# Проверка stock восстановлен
curl -s http://localhost:8080/products/<laptop_id> | jq .stock
# → 10 (восстановлено)
```

### 6. Бизнес-ошибки заказов

```bash
# Повторный заказ — у пользователя уже есть активный
curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d '{"items":[{"product_id":"<laptop_id>","quantity":1}]}' | jq .
# → 409: {"error_code":"ORDER_HAS_ACTIVE","message":"user already has an active order"}

# Чужой заказ — другой пользователь пытается посмотреть
curl -s http://localhost:8080/orders/<order_id> \
  -H "Authorization: Bearer $OTHER_USER_TOKEN" | jq .
# → 403: {"error_code":"ORDER_OWNERSHIP_VIOLATION","message":"order belongs to another user"}

# Недостаточно товара на складе
curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d '{"items":[{"product_id":"<laptop_id>","quantity":9999}]}' | jq .
# → 409: {"error_code":"INSUFFICIENT_STOCK","details":{"products":[{"product_id":"...","requested":9999,"available":10}]}}

# Заказ неактивного товара (ARCHIVED)
curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d '{"items":[{"product_id":"<archived_product_id>","quantity":1}]}' | jq .
# → 409: {"error_code":"PRODUCT_INACTIVE","message":"..."}

# Невалидный промокод
curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d '{"items":[{"product_id":"<laptop_id>","quantity":1}],"promo_code":"NONEXIST"}' | jq .
# → 422: {"error_code":"PROMO_CODE_INVALID","message":"..."}

# Сумма заказа ниже минимальной для промокода
curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d '{"items":[{"product_id":"<mouse_id>","quantity":1}],"promo_code":"FLAT500"}' | jq .
# → 422: {"error_code":"PROMO_CODE_MIN_AMOUNT","message":"..."}
```

### 7. Ролевая модель доступа

```bash
# SELLER пытается создать заказ — ЗАПРЕЩЕНО
curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -d '{"items":[{"product_id":"<product_id>","quantity":1}]}' | jq .
# → 403: {"error_code":"ACCESS_DENIED","message":"insufficient permissions"}

# USER пытается создать товар — ЗАПРЕЩЕНО
curl -s -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d '{"name":"Test","price":100,"stock":1,"category":"Test","status":"ACTIVE"}' | jq .
# → 403: {"error_code":"ACCESS_DENIED","message":"insufficient permissions"}

# USER пытается создать промокод — ЗАПРЕЩЕНО
curl -s -X POST http://localhost:8080/promo-codes \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d '{"code":"HACK","discount_type":"PERCENTAGE","discount_value":99,"min_order_amount":0,"max_uses":999,"valid_from":"2025-01-01T00:00:00Z","valid_until":"2027-01-01T00:00:00Z"}' | jq .
# → 403: {"error_code":"ACCESS_DENIED","message":"insufficient permissions"}

# Запрос без токена на защищённый эндпоинт
curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"items":[{"product_id":"<id>","quantity":1}]}' | jq .
# → 401: {"error_code":"TOKEN_INVALID","message":"missing authorization header"}

# GET /products без авторизации — OK (публичный эндпоинт)
curl -s http://localhost:8080/products | jq .
# → 200: список товаров
```

### 8. Проверка данных в БД

```bash
# Подключение к БД
make db-shell

# Посмотреть товары
SELECT id, name, price, stock, status, seller_id FROM products;

# Посмотреть заказы
SELECT id, user_id, status, total_amount, discount_amount, promo_code_id FROM orders;

# Посмотреть позиции заказа
SELECT oi.id, oi.order_id, p.name, oi.quantity, oi.price_at_order
FROM order_items oi JOIN products p ON oi.product_id = p.id;

# Посмотреть промокоды
SELECT code, discount_type, discount_value, current_uses, max_uses FROM promo_codes;

# Посмотреть пользователей
SELECT id, username, role, created_at FROM users;

# Посмотреть операции (rate limiting)
SELECT user_id, operation_type, created_at FROM user_operations ORDER BY created_at DESC;
```

## Ролевая модель

| Роль   | Описание                                        |
|--------|-------------------------------------------------|
| USER   | Покупатель — создаёт и управляет своими заказами |
| SELLER | Продавец — управляет своими товарами и промокодами |
| ADMIN  | Полный доступ ко всем операциям                  |

## Бизнес-логика заказов

- **Rate limiting** — ограничение частоты создания/обновления заказов
- **Активный заказ** — у пользователя может быть только один активный заказ (CREATED/PAYMENT_PENDING)
- **Резервирование stock** — при создании заказа остатки уменьшаются, при отмене — возвращаются
- **Снапшот цен** — цена фиксируется на момент заказа
- **Промокоды** — PERCENTAGE (до 70%) и FIXED_AMOUNT с проверкой минимальной суммы
- **Транзакционность** — все операции с заказами выполняются в одной транзакции

## Логирование

Каждый API-запрос логируется в JSON-формате:

```json
{
  "request_id": "uuid",
  "method": "POST",
  "endpoint": "/orders",
  "status_code": 201,
  "duration_ms": 15,
  "user_id": "uuid",
  "timestamp": "2026-03-04T18:00:00Z",
  "request_body": {"items": [...]}
}
```

- `X-Request-Id` пробрасывается в заголовке ответа
- Чувствительные данные (пароли, токены) маскируются в логах

## E2E-тестирование

Ниже приведены результаты ручного E2E-тестирования всех основных сценариев.

### 1. CRUD товаров

```bash
# Создание товара
curl -s -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{"name":"Laptop","price":1000,"stock":10,"category":"Electronics","status":"ACTIVE"}' | jq .
# → 201: {"id":"ec4ff149-...","name":"Laptop","price":1000,"stock":10,...}

curl -s -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{"name":"Mouse","price":50,"stock":5,"category":"Electronics","status":"ACTIVE"}' | jq .
# → 201: {"id":"78c3cd43-...","name":"Mouse","price":50,"stock":5,...}
```

### 2. Промокоды

```bash
# Создание промокода PERCENTAGE 10%
curl -s -X POST http://localhost:8080/promo-codes \
  -H "Content-Type: application/json" \
  -d '{"code":"SAVE10","discount_type":"PERCENTAGE","discount_value":10,"min_order_amount":100,"max_uses":5,"valid_from":"2025-01-01T00:00:00Z","valid_until":"2027-01-01T00:00:00Z"}' | jq .
# → 201: {"code":"SAVE10","discount_type":"PERCENTAGE","discount_value":10,...}
```

### 3. Создание заказа с промокодом

```bash
# Заказ: 2x Laptop (1000) + 1x Mouse (50) с промокодом SAVE10
curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "X-User-Id: 00000000-0000-0000-0000-000000000001" \
  -d '{"items":[{"product_id":"<laptop_id>","quantity":2},{"product_id":"<mouse_id>","quantity":1}],"promo_code":"SAVE10"}' | jq .
# → 201: total_amount=1845, discount_amount=205, status="CREATED"

# Проверка stock после заказа
curl -s http://localhost:8080/products/<laptop_id> | jq .stock
# → 8 (было 10, заказано 2)

curl -s http://localhost:8080/products/<mouse_id> | jq .stock
# → 4 (было 5, заказано 1)
```

### 4. Бизнес-ошибки заказов

```bash
# Повторный заказ — у пользователя уже есть активный
curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "X-User-Id: 00000000-0000-0000-0000-000000000001" \
  -d '{"items":[{"product_id":"<laptop_id>","quantity":1}]}' | jq .
# → 409: {"error_code":"ORDER_HAS_ACTIVE","message":"user already has an active order"}

# Чужой заказ — другой пользователь пытается посмотреть
curl -s http://localhost:8080/orders/<order_id> \
  -H "X-User-Id: 00000000-0000-0000-0000-000000000002" | jq .
# → 403: {"error_code":"ORDER_OWNERSHIP_VIOLATION","message":"order belongs to another user"}
```

### 5. Отмена заказа и возврат stock

```bash
# Отмена заказа
curl -s -X POST http://localhost:8080/orders/<order_id>/cancel \
  -H "X-User-Id: 00000000-0000-0000-0000-000000000001" | jq .
# → 200: status="CANCELED"

# Stock восстановлен
curl -s http://localhost:8080/products/<laptop_id> | jq .stock
# → 10 (восстановлено)

curl -s http://localhost:8080/products/<mouse_id> | jq .stock
# → 5 (восстановлено)
```

### 6. JWT-авторизация

```bash
# Регистрация пользователей с разными ролями
curl -s -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"buyer1","password":"pass123","role":"USER"}' | jq .
# → 201: {"id":"c33ca4b3-...","username":"buyer1","role":"USER"}

curl -s -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"seller1","password":"pass123","role":"SELLER"}' | jq .
# → 201: {"id":"7d2a7028-...","username":"seller1","role":"SELLER"}

curl -s -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"admin1","password":"pass123","role":"ADMIN"}' | jq .
# → 201: {"id":"b492cd9b-...","username":"admin1","role":"ADMIN"}

# Логин
curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"seller1","password":"pass123"}' | jq .
# → 200: {"access_token":"eyJ...","refresh_token":"eyJ..."}
```

### 7. Ролевая модель

```bash
# SELLER создаёт товар — OK
SELLER_TOKEN="<access_token продавца>"
curl -s -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -d '{"name":"Laptop","price":1000,"stock":10,"category":"Electronics","status":"ACTIVE"}' | jq .
# → 201: товар создан

# USER создаёт заказ — OK
BUYER_TOKEN="<access_token покупателя>"
curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d '{"items":[{"product_id":"<product_id>","quantity":1}]}' | jq .
# → 201: заказ создан

# SELLER пытается создать заказ — ЗАПРЕЩЕНО
curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -d '{"items":[{"product_id":"<product_id>","quantity":1}]}' | jq .
# → 403: {"error_code":"ACCESS_DENIED","message":"insufficient permissions"}

# Запрос без токена — ЗАПРЕЩЕНО
curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"items":[{"product_id":"<product_id>","quantity":1}]}' | jq .
# → 401: {"error_code":"TOKEN_INVALID","message":"missing authorization header"}

# GET /products без авторизации — OK (публичный)
curl -s http://localhost:8080/products | jq .
# → 200: список товаров
```

