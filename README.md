```markdown
# 🖥️ Screenshot Telegram Bot + Web Server

Эта программа написана на **Go** и совмещает в себе:
- Telegram-бота для удалённого получения скриншотов.
- HTTP-сервер для выдачи скриншотов через веб-запросы.
- Планировщик ежедневных скриншотов (в 00:30).
- Поддержку работы через **SOCKS5 proxy**.
- Систему логирования и ограничение доступа по `chat_id`.

---

## 🚀 Основные возможности

### 1. **Telegram Bot**

- Авторизация через флаг `--telegram-bot-token`.
- Поддержка режима отладки (`--debug`).
- Обработка входящих сообщений:
  - Если сообщение — это число (номер дисплея), бот делает скриншот и отправляет его.
  - Команда `/whoami` возвращает ваш `chat_id`.
- Проверка на разрешённый `chat_id` — посторонние пользователи игнорируются, а попытки доступа логируются.
- Поддержка работы через SOCKS5-прокси.

Пример логов:
2025/05/05 14:03:10 Received message from chat ID 123456789: 0
2025/05/05 14:03:11 Captured screen #0: 0_1920x1080.png
2025/05/05 14:03:12 Sent screenshot to user chat ID 123456789
```

```
### 2. **Веб-сервер**

- HTTP-сервер запускается на адресе, указанном флагом `--addr` (по умолчанию `:8080`).
- При GET-запросе вида: http://localhost:8080/?d=0

программа делает скриншот дисплея с номером `0` и возвращает PNG-изображение.
- Логирует IP-адрес каждого клиента, сделавшего запрос.

Пример логов:
2025/05/05 12:34:56 Received screenshot request from IP address: 192.168.1.42
2025/05/05 12:34:58 Captured screen #0: 0_1920x1080.png
```

```
### 3. **Ежедневные скриншоты (cron-like)**

- Если передан флаг `--daily`, бот автоматически делает скриншот в **00:30** каждый день и отправляет его в Telegram.
- Программа логирует, включена ли эта функция и через сколько времени произойдёт следующее выполнение.

Пример логов:
2025/05/05 00:00:01 Daily screenshot feature is enabled. The bot will send a screenshot every day at 00:30.
2025/05/05 00:00:01 Next daily screenshot will be sent in 30m0s (at 00:30).
2025/05/05 00:30:00 Sending daily screenshot for chat ID 123456789
2025/05/05 00:30:01 Successfully sent daily screenshot to chat ID 123456789
```

````
### 4. **Поддержка SOCKS5 Proxy**

Для работы через прокси используйте флаги или переменные окружения:

| Параметр | Описание |
|-----------|-----------|
| `--proxy-enable` | Включить использование прокси |
| `--proxy-socks5-addr` | Адрес SOCKS5-прокси (например, `127.0.0.1:9050`) |
| `--proxy-socks5-user` | Имя пользователя для аутентификации |
| `--proxy-socks5-password` | Пароль для аутентификации |

Пример запуска через Tor:
```bash
go run main.go \
  --telegram-bot-token="YOUR_TOKEN" \
  --allowed-chat-id=123456789 \
  --proxy-enable \
  --proxy-socks5-addr="127.0.0.1:9050"
````

---

## ⚙️ Конфигурация

Программа поддерживает **флаги командной строки** и **переменные окружения** (через `go-flags`).

| Параметр                  | Env               | Описание                     | По умолчанию   |
| ------------------------- | ----------------- | ---------------------------- | -------------- |
| `--telegram-bot-token`    | `TOKEN`           | Токен Telegram бота          | —              |
| `--debug`                 | `DEBUG`           | Включить режим отладки       | `false`        |
| `--addr`                  | `ADDR`            | Адрес и порт веб-сервера     | `:8080`        |
| `--daily`                 | `DAILY`           | Включить ежедневный скриншот | `false`        |
| `--allowed-chat-id`       | `ALLOWED_CHAT_ID` | Разрешённый chat_id          | (обязательный) |
| `--proxy-enable`          | `PROXY_ENABLE`    | Включить прокси              | `false`        |
| `--proxy-socks5-addr`     | `PROXY_ADDR`      | Адрес SOCKS5-прокси          | —              |
| `--proxy-socks5-user`     | `PROXY_USER`      | Логин для прокси             | —              |
| `--proxy-socks5-password` | `PROXY_PASSWORD`  | Пароль для прокси            | —              |

---

## 🧰 Примеры запуска

### Без прокси

```bash
go run main.go \
  --telegram-bot-token="YOUR_TOKEN" \
  --allowed-chat-id=123456789 \
  --daily \
  --addr=":9090"
```

### Через SOCKS5-прокси

```bash
go run main.go \
  --telegram-bot-token="YOUR_TOKEN" \
  --allowed-chat-id=123456789 \
  --proxy-enable \
  --proxy-socks5-addr="127.0.0.1:9050"
```

---

## 🧱 Установка и сборка

1. Установите зависимости:

   ```bash
   go mod tidy
   ```

2. Соберите бинарный файл:

   ```bash
   go build -o screenshot-bot main.go
   ```

3. Запустите программу:

   ```bash
   ./screenshot-bot --telegram-bot-token="YOUR_TOKEN" --allowed-chat-id=123456789
   ```

---

## 🔒 Безопасность

* Каждый запрос Telegram проверяется на соответствие разрешённому `chat_id`.
* Все попытки доступа от других пользователей логируются.
* IP-адреса всех запросов через веб-сервер фиксируются в логах.
* При необходимости можно ограничить доступ к HTTP-серверу (например, через фаервол).

---

## 🪵 Пример логов

```
2025/05/05 10:00:00 Authorized on account MyScreenshotBot
2025/05/05 10:00:05 Daily screenshot feature is enabled. The bot will send a screenshot every day at 00:30.
2025/05/05 10:01:00 Received message from chat ID 123456789: 0
2025/05/05 10:01:02 Captured screen #0: 0_1920x1080.png
2025/05/05 10:01:04 Successfully sent screenshot to chat ID 123456789
2025/05/05 10:02:00 Received screenshot request from IP address: 192.168.0.45
2025/05/05 10:02:02 Captured screen #1: 1_2560x1440.png
2025/05/05 10:02:03 Sent screenshot to web client
```

---

## 🧩 Зависимости

* [Go Telegram Bot API](https://github.com/go-telegram-bot-api/telegram-bot-api)
* [Kbinani Screenshot](https://github.com/kbinani/screenshot)
* [Go Flags (umputun)](https://github.com/umputun/go-flags)
* [golang.org/x/net/proxy](https://pkg.go.dev/golang.org/x/net/proxy)

---

## 🧠 Примечания

* Скриншоты создаются с помощью системных API, поэтому программа должна запускаться **на машине с GUI**.
* Если указан неверный номер дисплея, программа выполнит до 5 попыток захвата.
* Время сервера используется для расчёта расписания `--daily`.
* Программа многопоточная — Telegram, Web и Daily-функции работают независимо друг от друга.
