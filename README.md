# ИИ-бот учёта семейных расходов

Сервис учёта расходов: Telegram-бот с LLM (естественный язык) и веб-панель для просмотра/редактирования трат всех пользователей.

## Состав

| Сервис | Назначение |
|---|---|
| `postgres` | БД |
| `bot` | Go: Telegram webhook, LLM-агент, REST API |
| `front` | Веб-панель (Vite → nginx) |

```
services/
  bot/     # backend + telegram
  front/   # web UI
deploy/    # docker compose, .env
```

## Принятые решения

- **Семья** не моделируется отдельно: «семья» = все пользователи, которые писали боту. Веб без авторизации видит всех.
- **Бот** работает только со своими тратами (`user_id` из Telegram).
- **Валюта:** RUB (₽), отдельной колонки валюты нет.
- **Таймзона:** `Europe/Moscow` (даты «сегодня» / «вчера», периоды «неделя» / «месяц»).
- **Категории (enum):** еда, транспорт, жильё, развлечения, здоровье, покупки, другое.
- **«Последняя» трата** (фразы вроде «последнее было не 350, а 450») определяется по **`created_at DESC`** — момент записи в систему, а не по `spent_at`.
- **Неделя** в фильтрах: с понедельника по текущий день (Moscow).
- **Webhook и REST на разных портах:** webhook рассчитан на туннель ([Serveo](https://serveo.net/) и т.п.), API слушает отдельно.
- **LLM:** OpenAI-compatible API через [OpenRouter](https://openrouter.ai/openrouter/free) (`openrouter/free` по умолчанию), ключ и модель — из конфига/env.
- **Архитектура backend:** один Go-сервис (общий репозиторий трат); бот вызывает LLM tools → repo; веб ходит в REST.

## Быстрый старт (Docker)

1. Скопируйте env:

```bash
cp deploy/.env.example deploy/.env
```

2. Заполните в `deploy/.env` как минимум:

- `TELEGRAM_BOT_TOKEN`
- `LLM_API_KEY`
- `WEBHOOK_PUBLIC_URL` — публичный HTTPS URL туннеля на порт webhook
- при необходимости `VITE_API_URL` (для браузера, обычно `http://localhost:8080`)

3. Поднимите стек:

```bash
cd deploy
docker compose up -d --build
```

4. Пробросьте webhook (пример Serveo):

```bash
ssh -R 80:localhost:3000 serveo.net
```

URL из Serveo укажите в `WEBHOOK_PUBLIC_URL`, перезапустите bot при смене URL.

### Порты по умолчанию

| Порт (host) | Сервис |
|---|---|
| `5432` / `POSTGRES_PORT` | Postgres |
| `3000` / `BOT_PORT` | Telegram webhook |
| `8080` / `API_PORT` | REST API |
| `5173` / `FRONT_PORT` | Веб-панель |

- Панель: http://localhost:5173  
- Health: http://localhost:8080/api/health  

`VITE_API_URL` задаётся **на этапе сборки** образа `front`. После изменения — пересоберите front.

## Переменные окружения

См. `deploy/.env.example`. Все значения для compose берутся из `deploy/.env`.

Важное:

| Переменная | Смысл |
|---|---|
| `DATABASE_URL` | В Docker — хост `postgres`; локально с хоста — `localhost` |
| `MIGRATIONS_DIR` | В контейнере: `/app/migrations` |
| `WEBHOOK_LISTEN` / `API_LISTEN` | Адреса слушателей внутри bot |
| `LLM_BASE_URL` / `LLM_MODEL` | Провайдер и модель |

## REST API (кратко)

Базовый URL: `http://localhost:8080`

| Method | Path | Описание |
|---|---|---|
| GET | `/api/health` | healthcheck |
| GET | `/api/expenses` | список; query: `user_id`, `category`, `from`, `to`, `limit` |
| GET | `/api/expenses/sum` | сумма по тем же фильтрам (без `limit`) |
| GET | `/api/expenses/{id}` | одна трата |
| POST | `/api/expenses` | создать |
| PUT/PATCH | `/api/expenses/{id}` | изменить |
| DELETE | `/api/expenses/{id}` | удалить |

Авторизация не требуется.

## Telegram-бот

Без кнопок и команд: обычный текст → LLM (tool calling) → CRUD.

Примеры:

- «кофе 350», «такси 1200 вчера»
- «покажи траты за неделю»
- «сколько потратил на еду в этом месяце»
- «последнее было не 350, а 450»
- «удали такси»
