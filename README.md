[![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg?style=for-the-badge)](https://opensource.org/licenses/MIT)
[![GitHub last commit](https://img.shields.io/github/last-commit/nazgard/go-tg-screenshot-bot?style=for-the-badge)](https://github.com/nazgard/go-tg-screenshot-bot)
[![Tests](https://github.com/nazgard/go-tg-screenshot-bot/actions/workflows/go-test.yml/badge.svg)](https://github.com/nazgard/go-tg-screenshot-bot/actions/workflows/go-test.yml)
[![Codecov](https://codecov.io/gh/nazgard/go-tg-screenshot-bot/graph/badge.svg)](https://codecov.io/gh/nazgard/go-tg-screenshot-bot)

# 🖥️ Screenshot Bot

Удалённый доступ к скриншотам экрана через **Telegram-бота**, **веб-сервер** и **ежедневную автоматическую рассылку**.

Поддержка нескольких мониторов, SOCKS5-прокси, Basic Auth, graceful shutdown и подробное логирование.

Идеально для домашнего ПК, удалённого рабочего стола или VPS с GUI.

---

## ⚡ Быстрый старт

```bash
go run main.go \
  --telegram-bot-token="123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11" \
  --allowed-chat-id=123456789
```

Теперь:
- Напишите боту любое число (например `0`) → получите скриншот соответствующего дисплея.
- Откройте в браузере: `http://your-ip:8080/?d=0` → скачаете PNG-скриншот.
- Используйте команду `/whoami`, чтобы узнать свой chat ID.

---

## 🚀 Основные возможности

### 1. Telegram-бот
- Авторизация по одному разрешённому `chat_id`.
- Команда `/whoami` — возвращает ваш chat ID.
- Отправка числа → скриншот указанного дисплея (0, 1, 2…).
- Режим отладки (`--debug`).
- Все действия логируются.

### 2. Веб-сервер
- Эндпоинт: `GET /?d=<номер_дисплея>`
- Возвращает PNG-изображение напрямую.
- Опциональная **Basic Authentication** (`--auth-enable`).
- Логирует IP каждого клиента.

**Примеры запросов:**

```bash
# Без авторизации
curl "http://localhost:8080/?d=0" -o screen.png

# С Basic Auth
curl -u admin:supersecret "http://localhost:8080/?d=1" -o screen.png
```

### 3. Ежедневные скриншоты
- Флаг `--daily` включает отправку скриншота **каждый день в 00:30**.
- Используется последний запрошенный дисплей.
- Логируется время следующей отправки и результат.

### 4. SOCKS5-прокси
Полная поддержка прокси для всех исходящих запросов Telegram API (Tor, корпоративный прокси и т.д.).

---

## ⚙️ Конфигурация

| Параметр                     | Env переменная       | Обязательный? | По умолчанию | Описание                                      |
|------------------------------|----------------------|---------------|--------------|-----------------------------------------------|
| `--telegram-bot-token`       | `TOKEN`              | Да            | —            | Токен Telegram-бота                           |
| `--allowed-chat-id`          | `ALLOWED_CHAT_ID`    | Да            | —            | Разрешённый chat ID                           |
| `--addr`                     | `ADDR`               | Нет           | `:8080`      | Адрес и порт веб-сервера                      |
| `--daily`                    | `DAILY`              | Нет           | `false`      | Ежедневный скриншот в 00:30                    |
| `--debug`                    | `DEBUG`              | Нет           | `false`      | Режим отладки Telegram API                    |
| `--auth-enable`              | `AUTH_ENABLE`        | Нет           | `false`      | Включить Basic Auth для веб-сервера           |
| `--auth-user`                | `AUTH_USER`          | Нет*          | —            | Логин Basic Auth                              |
| `--auth-password`            | `AUTH_PASS`          | Нет*          | —            | Пароль Basic Auth                             |
| `--proxy-enable`             | `PROXY_ENABLE`       | Нет           | `false`      | Включить SOCKS5-прокси                        |
| `--proxy-socks5-addr`        | `PROXY_ADDR`         | Нет*          | —            | Адрес прокси (например, `127.0.0.1:9050`)      |
| `--proxy-socks5-user`        | `PROXY_USER`         | Нет           | —            | Логин прокси (опционально)                    |
| `--proxy-socks5-password`    | `PROXY_PASSWORD`     | Нет           | —            | Пароль прокси (опционально)                   |

_* Обязательны только при включении соответствующей функции._

---

## 🧰 Примеры запуска

### Минимальный
```bash
./go-tg-screenshot-bot --telegram-bot-token="YOUR_TOKEN" --allowed-chat-id=123456789
```

### С ежедневной отправкой и веб-сервером на другом порту
```bash
./go-tg-screenshot-bot \
  --telegram-bot-token="YOUR_TOKEN" \
  --allowed-chat-id=123456789 \
  --daily \
  --addr=":9090"
```

### С Basic Auth
```bash
./go-tg-screenshot-bot \
  --telegram-bot-token="YOUR_TOKEN" \
  --allowed-chat-id=123456789 \
  --auth-enable \
  --auth-user="admin" \
  --auth-password="supersecret123"
```

### Через Tor (SOCKS5)
```bash
./go-tg-screenshot-bot \
  --telegram-bot-token="YOUR_TOKEN" \
  --allowed-chat-id=123456789 \
  --proxy-enable \
  --proxy-socks5-addr="127.0.0.1:9050"
```

---

## 🧱 Сборка и установка

```bash
# Клонирование и зависимости
git clone https://github.com/nazgard/go-tg-screenshot-bot.git
cd go-tg-screenshot-bot
go mod tidy

# Сборка
go build -o go-tg-screenshot-bot main.go

# Запуск
./go-tg-screenshot-bot --telegram-bot-token="YOUR_TOKEN" --allowed-chat-id=123456789
```

---

## ⚠️ Совместимость

- **Linux**: Требуется X11 (GNOME, KDE, XFCE, i3 и т.д.). Работает на большинстве дистрибутивов.
- **Windows**: Полная поддержка всех мониторов.
- **macOS**: Только основной дисплей (требуется разрешение на запись экрана).

> На headless-серверах без GUI программа **не работает**.

---

## 🔒 Безопасность

- Telegram: доступ только с одного разрешённого `chat_id`.
- Веб: опциональная Basic Auth + логирование IP.
- Рекомендации для продакшена:
    - Всегда включайте `--auth-enable`.
    - Ограничьте доступ к порту фаерволом (ufw/iptables).
    - Используйте reverse-прокси (nginx, Caddy, Traefik) с HTTPS.
    - **Basic Auth не шифрует данные** — без HTTPS credentials передаются в открытом виде.

---

## 🪵 Пример логов

```
2026/01/06 12:00:00 Authorized on account MyScreenshotBot
2026/01/06 12:00:00 Starting web server on :8080
2026/01/06 12:00:00 Daily screenshot enabled — sending at 00:30 every day.
2026/01/06 12:05:12 Received message from chat ID 123456789: 0
2026/01/06 12:05:13 Captured screen #0: 0_1920x1080.png
2026/01/06 12:10:22 Received screenshot request from IP: 192.168.1.100
2026/01/06 12:10:23 Captured screen #1: 1_2560x1440.png
```

---

## 🧩 Зависимости

- `github.com/go-telegram-bot-api/telegram-bot-api/v5`
- `github.com/kbinani/screenshot`
- `github.com/umputun/go-flags`
- `golang.org/x/net/proxy`

---

## 📄 OpenAPI (Swagger)

API документирован с помощью OpenAPI 3.0.

- Файл спецификации: [`openapi.yaml`](swagger.yaml)
- Онлайн-просмотр:
    - [Swagger Editor](https://editor.swagger.io/?url=https://raw.githubusercontent.com/nazgard/go-tg-screenshot-bot/main/swagger.yaml)
    - [Redoc](https://redocly.github.io/redoc/?url=https://raw.githubusercontent.com/nazgard/go-tg-screenshot-bot/main/swagger.yaml)

---

**Лицензия:** MIT  
**Автор:** nazgard  
**Приятного использования!** 🚀
