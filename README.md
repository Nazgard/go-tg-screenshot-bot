# Описание программы

Эта программа написана на языке Go и выполняет функции веб-сервера и бота для Telegram. Программа позволяет делать скриншоты экранов и отправлять их либо в ответ на HTTP-запросы, либо в виде изображений в чате Telegram. Программа поддерживает многопоточную обработку запросов и использует стандартные библиотеки Go вместе с библиотеками сторонних разработчиков.

## Основные функции программы:

### 1. **Telegram Bot**
   - Программа реализует бота для Telegram, который авторизуется с помощью токена, переданного через флаг командной строки `--token`.
   - Бот прослушивает входящие сообщения от пользователей и ожидает, что в сообщении будет содержаться номер дисплея (экранного монитора).
   - После получения сообщения бот делает скриншот экрана с указанным номером и отправляет его обратно пользователю в виде изображения в чате.

### 2. **Веб-сервер**
   - Веб-сервер слушает HTTP-запросы на порту `8080`.
   - При обращении к серверу с GET-запросом на путь `/` и передачей параметра `d` (номер дисплея), программа делает скриншот указанного экрана.
   - Скриншот возвращается в виде изображения в формате PNG в ответе на HTTP-запрос.

### 3. **Общие функциональные возможности**
   - Программа использует библиотеку `github.com/kbinani/screenshot` для захвата скриншотов с разных дисплеев.
   - В случае возникновения ошибок при захвате скриншота, программа предпринимает повторные попытки (до 5 раз), чтобы сделать успешный скриншот.
   - Программа разделяет обработку запросов Telegram и HTTP на разные горутины, что позволяет ей обрабатывать сообщения и запросы одновременно.

## Использование программы:

1. **Запуск программы:**
   - Для запуска необходимо указать токен Telegram бота с помощью флага `--token`.
   - Пример запуска:
     ```bash
     go run main.go --token="YOUR_TELEGRAM_BOT_TOKEN"
     ```

2. **Использование Telegram бота:**
   - Отправьте сообщение боту, содержащее номер дисплея (например, "0" для первого дисплея).
   - Бот ответит вам сообщением, содержащее изображение — скриншот указанного дисплея.

3. **Использование веб-сервера:**
   - Откройте браузер и перейдите по адресу `http://localhost:8080/?d=0`, где `d=0` — номер дисплея.
   - В ответ вы получите изображение в формате PNG, содержащее скриншот указанного дисплея.

### Примечания:
- В случае возникновения ошибок, связанных с захватом изображения (например, если дисплей не существует), программа попытается сделать скриншот до 5 раз перед тем, как отказаться от попытки.
- Программа работает в многопоточном режиме, что обеспечивает её эффективность при одновременной обработке запросов Telegram и HTTP.