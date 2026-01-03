# 🖥️ Screenshot Telegram Bot + Web Server

Программа на **Go**, которая позволяет удалённо получать скриншоты экрана через:

- Telegram-бота  
- HTTP-сервер с опциональной **Basic Authentication**  
- Автоматическую ежедневную отправку скриншота в 00:30  

Поддерживается работа через **SOCKS5-прокси** и подробное логирование всех действий.

---

## 🚀 Основные возможности

### 1. Telegram Bot

- Авторизация по токену Telegram.
- Режим отладки (`--debug`).
- Обработка сообщений:
  - Отправьте число — бот сделает скриншот указанного дисплея и пришлёт фото.
  - Команда `/whoami` — возвращает ваш `chat_id`.
- Доступ ограничен одним разрешённым `chat_id` (попытки от других пользователей игнорируются и логируются).

### 2. Веб-сервер с Basic Auth

- Запускается на порту, указанном в `--addr` (по умолчанию `:8080`).
- Эндпоинт: `GET /?d=<номер_дисплея>`
- Возвращает PNG-изображение скриншота.
- **Опциональная Basic Authentication** — включается флагом `--auth-enable`.
- Логирует IP-адрес каждого клиента.

**Примеры запросов:**

```bash
# Без авторизации (если auth отключена)
curl "http://localhost:8080/?d=0" -o screen.png

# С Basic Auth (если включена)
curl -u username:password "http://localhost:8080/?d=0" -o screen.png
```

### 3. Ежедневные скриншоты

- При включении `--daily` бот каждый день в **00:30** отправляет скриншот последнего использованного дисплея в разрешённый чат.
- Логируется время следующей отправки и результат выполнения.

### 4. SOCKS5 Proxy

Поддержка прокси для всех исходящих соединений Telegram API (удобно при работе через Tor или корпоративный прокси).

---

## ⚙️ Конфигурация

Программа использует **флаги командной строки** и **переменные окружения**.

| Параметр                     | Env                     | Описание                                      | По умолчанию    |
|------------------------------|-------------------------|-----------------------------------------------|-----------------|
| `--telegram-bot-token`       | `TOKEN`                 | Токен Telegram бота                           | — (обязательный)|
| `--debug`                    | `DEBUG`                 | Режим отладки Telegram                        | `false`         |
| `--addr`                     | `ADDR`                  | Адрес веб-сервера                             | `:8080`         |
| `--daily`                    | `DAILY`                 | Включить ежедневный скриншот в 00:30           | `false`         |
| `--allowed-chat-id`          | `ALLOWED_CHAT_ID`       | Разрешённый Telegram chat ID                   | — (обязательный)|
| `--auth-enable`              | `AUTH_ENABLE`           | Включить Basic Auth для веб-сервера           | `false`         |
| `--auth-user`                | `AUTH_USER`             | Логин для Basic Auth                          | —               |
| `--auth-password`            | `AUTH_PASS`             | Пароль для Basic Auth                         | —               |
| `--proxy-enable`             | `PROXY_ENABLE`          | Включить SOCKS5-прокси                        | `false`         |
| `--proxy-socks5-addr`        | `PROXY_ADDR`            | Адрес прокси (например, `127.0.0.1:9050`)      | —               |
| `--proxy-socks5-user`        | `PROXY_USER`            | Логин прокси (опционально)                    | —               |
| `--proxy-socks5-password`    | `PROXY_PASSWORD`        | Пароль прокси (опционально)                   | —               |

---

## 🧰 Примеры запуска

### Базовый запуск

```bash
go run main.go \
  --telegram-bot-token="YOUR_TOKEN" \
  --allowed-chat-id=123456789 \
  --daily \
  --addr=":9090"
```

### С Basic Authentication

```bash
go run main.go \
  --telegram-bot-token="YOUR_TOKEN" \
  --allowed-chat-id=123456789 \
  --auth-enable \
  --auth-user="admin" \
  --auth-password="supersecret123"
```

Запрос с авторизацией:
```bash
curl -u admin:supersecret123 "http://localhost:8080/?d=0" -o screen.png
```

### Через SOCKS5-прокси (например, Tor)

```bash
go run main.go \
  --telegram-bot-token="YOUR_TOKEN" \
  --allowed-chat-id=123456789 \
  --proxy-enable \
  --proxy-socks5-addr="127.0.0.1:9050"
```

### Полный набор опций

```bash
go run main.go \
  --telegram-bot-token="YOUR_TOKEN" \
  --allowed-chat-id=123456789 \
  --daily \
  --auth-enable \
  --auth-user="user" \
  --auth-password="pass" \
  --proxy-enable \
  --proxy-socks5-addr="127.0.0.1:9050"
```

---

## 🧱 Установка и сборка

```bash
# Подготовка зависимостей
go mod tidy

# Сборка бинарника
go build -o screenshot-bot main.go

# Запуск
./screenshot-bot --telegram-bot-token="YOUR_TOKEN" --allowed-chat-id=123456789
```

---

## 🔒 Безопасность

- **Telegram**: доступ только с одного разрешённого `chat_id`.
- **Веб-сервер**: опциональная **Basic Authentication**. Рекомендуется включать в продакшене.
- Все веб-запросы логируют IP-адрес клиента.
- Для повышенной безопасности:
    - Ограничьте доступ к порту через фаервол.
    - Используйте reverse-прокси (nginx/Caddy) с HTTPS.
    - Всегда включайте `--auth-enable`.

> **Важно**: Basic Auth передаёт данные в Base64 (не шифруется). В открытой сети обязательно используйте HTTPS!

---

## 🪵 Пример логов

```
2026/01/03 10:00:00 Authorized on account MyScreenshotBot
2026/01/03 10:00:00 Daily screenshot feature is enabled...
2026/01/03 10:05:12 Received message from chat ID 123456789: 0
2026/01/03 10:05:13 Captured screen #0: 0_1920x1080.png
2026/01/03 10:10:22 Received screenshot request from IP address: 192.168.1.100
2026/01/03 10:10:23 Captured screen #1: 1_2560x1440.png
```

---

## 🧩 Зависимости

- `github.com/go-telegram-bot-api/telegram-bot-api/v5`
- `github.com/kbinani/screenshot`
- `github.com/umputun/go-flags`
- `golang.org/x/net/proxy`

---

Программа предназначена для запуска на машинах с графическим интерфейсом (домашний ПК, VPS с X11/VNC и т.д.).
