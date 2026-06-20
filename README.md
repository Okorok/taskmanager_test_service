## Архитектура

```
cmd/service            — точка входа, сборка зависимостей, роутинг
internal/
  config               — загрузка YAML-конфига
  entity               — доменные сущности (db-теги)
  infrastructure       — UnitOfWork (транзакции через context), JWT, bcrypt,
                         ошибки, Redis-кеш, testcontainers-хелпер
  repository           — доступ к MySQL (sqlx), tx-aware runner, сложные SQL
  service              — бизнес-логика, команды, доменные ошибки, проверки прав
  http/
    middleware         — JWT-auth, recovery, chain
    web                — общие JSON-ответы и маппинг доменных ошибок в HTTP-коды
    *_handler          — HTTP-хендлеры (auth, team, task, analytics)
```

## Запуск

```bash
make docker-up
```

Поднимаются `db` (MySQL, `:3306`), `redis` (`:6379`), одноразовый `migrate`
(применяет `migrations/001_init_schema.up.sql`) и `app` (`:8080`).

Доступы к MySQL: пользователь/пароль/БД — `app/app/app`.

## Тесты

```bash
make test-unit          # unit-тесты 
make test-integration   # интеграционные тесты
make test               # все тесты
make generate           # перегенерировать моки (mockery, конфиг .mockery.yaml)
```

```bash
go test -cover ./internal/service/...
```

## Аутентификация

Все ручки, кроме `/register` и `/login`, требуют заголовок:

```http
Authorization: Bearer <jwt>
```

Токен выдаётся ручкой `/login` и живёт 24 часа (настраивается в конфиге).

## API

### Аутентификация

```bash
# Регистрация
curl -X POST localhost:8080/api/v1/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"secret123","name":"Alice"}'

# Логин -> {"token":"..."}
curl -X POST localhost:8080/api/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"secret123"}'
```

### Команды

```bash
# Создать команду (создатель становится owner)
curl -X POST localhost:8080/api/v1/teams \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"Platform"}'

# Список команд, где я состою
curl localhost:8080/api/v1/teams -H "Authorization: Bearer $TOKEN"

# Пригласить пользователя (только owner/admin), роль: admin|member
curl -X POST localhost:8080/api/v1/teams/1/invite \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"user_id":2,"role":"member"}'
```

### Задачи

```bash
# Создать задачу (только член команды)
curl -X POST localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"team_id":1,"title":"Fix bug","description":"...","priority":"high","assignee_id":2}'

# Список с фильтрами и пагинацией
curl "localhost:8080/api/v1/tasks?team_id=1&status=todo&assignee_id=2&limit=20&offset=0" \
  -H "Authorization: Bearer $TOKEN"

# Обновить задачу (частично; проверка прав)
curl -X PUT localhost:8080/api/v1/tasks/5 \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"status":"done"}'

# История изменений задачи
curl localhost:8080/api/v1/tasks/5/history -H "Authorization: Bearer $TOKEN"

# Комментарии
curl -X POST localhost:8080/api/v1/tasks/5/comments \
  -H "Authorization: Bearer $TOKEN" -d '{"body":"looks good"}'
curl localhost:8080/api/v1/tasks/5/comments -H "Authorization: Bearer $TOKEN"
```

### Аналитика (сложные SQL-запросы)

```bash
# (а) JOIN 3+ таблиц + агрегация: по каждой команде — участники и done-задачи за 7 дней
curl localhost:8080/api/v1/analytics/team-stats -H "Authorization: Bearer $TOKEN"

# (б) оконная функция ROW_NUMBER(): топ-3 автора задач в каждой команде за месяц
curl localhost:8080/api/v1/analytics/top-creators -H "Authorization: Bearer $TOKEN"

# (в) условие по связанным таблицам: задачи, где assignee не член команды задачи
curl localhost:8080/api/v1/analytics/inconsistent-tasks -H "Authorization: Bearer $TOKEN"
```

## Реализация требований ТЗ

### Схема БД (6 таблиц, 10 связей)

`users`, `teams`, `team_members` (M:N + роль), `tasks`, `task_history`, `task_comments`.
Все внешние ключи из ТЗ заданы в `migrations/001_init_schema.up.sql`.

### Сложные SQL-запросы

Реализованы в `internal/repository/analytics_repository.go`:
- **(а)** `TeamStats` — `LEFT JOIN` teams+team_members+tasks с `COUNT(DISTINCT ...)`
  и условной агрегацией по статусу/окну в 7 дней.
- **(б)** `TopCreatorsPerTeam` — подзапрос с `ROW_NUMBER() OVER (PARTITION BY team ...)`,
  фильтр `rnk <= 3`.
- **(в)** `InconsistentTasks` — `NOT EXISTS` подзапрос по `team_members` (валидация целостности).

Покрыты интеграционными тестами на реальной MySQL.

### Оптимизация

- **Кеш Redis**: списки задач команды кешируются на 5 минут
  (`internal/infrastructure/cache`); при создании/обновлении задачи кеш команды
  инвалидируется (`SCAN` + `DEL` по префиксу `tasks:team:{id}:`).
- **Индексы MySQL** (см. миграцию):
  - `tasks(team_id, status, id)` — основной фильтр списка + пагинация по id;
  - `tasks(assignee_id)` — фильтр по исполнителю;
  - `tasks(team_id, created_at)` — аналитика по периоду;
  - `team_members(user_id)`, `task_history(task_id, changed_at)`, `task_comments(task_id, created_at)`,
    уникальный `users(email)`.
- **Connection pooling**: `SetMaxOpenConns/SetMaxIdleConns/SetConnMaxLifetime` (конфиг `db.*`).
- **Пагинация на уровне БД**: `LIMIT ? OFFSET ?` (по умолчанию limit=20, максимум 100).

### Консистентность

- Создание команды + добавление owner — в одной транзакции.
- Создание/обновление задачи + записи в `task_history` — в одной транзакции (UnitOfWork).
- Целостность исполнителя: при назначении проверяется членство в команде;
  запрос (в) ловит расхождения, которые могли возникнуть исторически.
