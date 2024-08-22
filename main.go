package main

import (
	"bytes"
	"flag"
	"fmt"
	"image/png"
	"log"
	"net/http"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kbinani/screenshot"
)

// Структура application содержит конфигурацию и экземпляр бота Telegram.
type application struct {
	config struct {
		token   string // Токен для Telegram API
		debug   bool   // Режим отладки
		webPort string // Порт для веб-сервера
	}
	tgBot *tgbotapi.BotAPI // Экземпляр бота Telegram
}

func main() {
	// Конфигурация приложения
	app := configure()

	// Запуск прослушивания сообщений Telegram в отдельной горутине
	go listenTg(app)

	// Запуск веб-сервера в отдельной горутине
	go listenWeb(app)

	// Блокировка основного потока, чтобы программа продолжала работать
	select {}
}

// configure настраивает приложение и возвращает его экземпляр.
func configure() *application {
	// Определение флагов командной строки
	token := flag.String("token", "Token", "Enter telegram token")
	debug := flag.Bool("debug", false, "Debug mode")
	port := flag.String("port", "8080", "Web port")
	flag.Parse()

	// Создание нового бота Telegram
	bot, err := tgbotapi.NewBotAPI(*token)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = *debug
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Возврат настроенного экземпляра приложения
	return &application{
		config: struct {
			token   string
			debug   bool
			webPort string
		}{
			token:   *token,
			debug:   *debug,
			webPort: *port,
		},
		tgBot: bot,
	}
}

// listenTg обрабатывает входящие сообщения Telegram.
func listenTg(app *application) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// Получаем канал для чтения обновлений от Telegram
	updates := app.tgBot.GetUpdatesChan(u)

	// Обработка каждого обновления
	for update := range updates {
		if update.Message == nil {
			return
		}

		// Запускаем обработку сообщения в новой горутине
		go func() {
			disNum, _ := strconv.Atoi(update.Message.Text) // Преобразование текста сообщения в номер дисплея
			fileName, buf := screen(disNum)                // Захват экрана

			// Подготовка сообщения с изображением для отправки
			msg := tgbotapi.PhotoConfig{
				BaseFile: tgbotapi.BaseFile{
					BaseChat: tgbotapi.BaseChat{
						ChatID: update.Message.From.ID,
					},
					File: tgbotapi.FileReader{
						Name:   fileName,
						Reader: buf,
					},
				},
			}
			app.tgBot.Send(&msg) // Отправка сообщения
		}()
	}
}

// screen выполняет захват экрана для указанного дисплея и возвращает имя файла и буфер с изображением.
func screen(disNum int) (string, *bytes.Buffer) {
	bounds := screenshot.GetDisplayBounds(disNum) // Получение границ дисплея
	img, err := screenshot.CaptureRect(bounds)    // Захват изображения экрана
	if err != nil {
		// Попытка повторного захвата экрана до 5 раз при ошибке
		for i := 0; i < 5; i++ {
			fmt.Printf("Error while capture rect %s. Trying again\n", err.Error())
			time.Sleep(1 * time.Second)
			bounds = screenshot.GetDisplayBounds(disNum)
			img, err = screenshot.CaptureRect(bounds)
			if err == nil {
				break
			}
		}
		if err != nil {
			fmt.Printf("Error while capture rect %s\n", err.Error())
			return "", nil
		}
	}
	// Формирование имени файла
	fileName := fmt.Sprintf("%d_%dx%d.png", disNum, bounds.Dx(), bounds.Dy())
	var buf bytes.Buffer
	png.Encode(&buf, img) // Кодирование изображения в PNG и запись в буфер
	fmt.Printf("#%d : %v \"%s\"\n", disNum, bounds, fileName)
	return fileName, &buf
}

// listenWeb запускает веб-сервер и обрабатывает HTTP-запросы.
func listenWeb(app *application) {
	// Обработчик для главной страницы
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		displayNumberStr := r.URL.Query().Get("d")  // Получение параметра "d" из запроса
		disNum, _ := strconv.Atoi(displayNumberStr) // Преобразование параметра в номер дисплея
		_, buf := screen(disNum)                    // Захват экрана

		// Устанавливаем заголовки ответа
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", strconv.Itoa(len(buf.Bytes())))

		// Отправляем изображение в ответе
		if _, err := w.Write(buf.Bytes()); err != nil {
			log.Printf("Failed to write image to response: %v", err)
		}
	})

	// Запуск веб-сервера на указанном порту
	fmt.Printf("Starting web server on %s", app.config.webPort)
	if err := http.ListenAndServe(":"+app.config.webPort, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
