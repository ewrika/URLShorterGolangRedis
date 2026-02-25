### README

**UrlShorter**

Простой сервис сокращения ссылок с rate limiting, написанный на Go и запускаемый через Docker Compose.

---

### Что делает проект

- Принимает длинный URL и возвращает короткий вид `http://DOMAIN/<id>`.
- Если такой URL уже сокращён, возвращает уже существующую короткую ссылку.
- Ограничивает количество запросов сокращения по IP (rate limiting).
- При переходе по короткой ссылке делает HTTP 301‑редирект на исходный URL.
- Ведёт счётчик переходов по сокращённым ссылкам.

---

### Стек и технологии

- **Язык**: Go
- **Web‑фреймворк**: [Fiber](https://gofiber.io/)
- **База**: Redis
- **Контейнеризация**: Docker, Docker Compose
- **Прочее**:
  - `github.com/asaskevich/govalidator` – валидация URL
  - `github.com/google/uuid` – генерация коротких ID
  - `github.com/joho/godotenv` – загрузка `.env`

---

### Основные эндпоинты

- **POST `/api/v1`** – сократить ссылку  
  - Тело запроса (JSON):
    ```json
    {
      "url": "https://example.com",
      "short": "",
      "expiry": 24
    }
    ```
    - `url` – исходный URL (обязателен)
    - `short` – свой кастомный код (опционально)
    - `expiry` – срок жизни в часах (по умолчанию 24)
  - Ответ (пример):
    ```json
    {
      "url": "https://example.com",
      "short": "http://localhost:3000/abc123",
      "expiry": 24,
      "rate_limit": 9,
      "rate_limit_reset": 30
    }
    ```

- **GET `/:url`** – переход по сокращённой ссылке  
  - Пример: `GET http://localhost:3000/abc123`  
  - Возвращает 301 Redirect на исходный URL или 404, если код не найден.

---

### Переменные окружения (`api/.env`)

Пример:

```env
REDIS_URL="db:6379"
REDIS_PASSWORD=""
PORT=":3000"
DOMAIN="localhost:3000"
API_QUOTA=10
```

- `REDIS_URL` – адрес Redis (в Docker‑сети это сервис `db`).
- `PORT` – порт, который слушает API (Fiber).
- `DOMAIN` – домен/хост, используемый при формировании коротких ссылок.
- `API_QUOTA` – количество запросов сокращения в окне rate limiting.

---

### Запуск через Docker Compose

В корне проекта:

```bash
docker-compose up -d --build
```

После запуска:

- API доступен по адресу `http://localhost:3000`.
- Redis доступен на `localhost:6379`.
