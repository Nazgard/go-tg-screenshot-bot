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

type application struct {
	config struct {
		token   string
		debug   bool
		webPort string
	}
	tgBot *tgbotapi.BotAPI
}

func main() {
	app := configure()

	go listenTg(app)
	go listenWeb(app)

	select {}
}

func configure() *application {
	token := flag.String("token", "Token", "Enter telegram token")
	debug := flag.Bool("debug", false, "Debug mode")
	port := flag.String("port", "8080", "Web port")
	flag.Parse()
	bot, err := tgbotapi.NewBotAPI(*token)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = *debug
	log.Printf("Authorized on account %s", bot.Self.UserName)

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

func listenTg(app *application) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := app.tgBot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			return
		}
		go func() {
			disNum, _ := strconv.Atoi(update.Message.Text)
			fileName, buf := screen(disNum)

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
			app.tgBot.Send(&msg)
		}()
	}
}

func screen(disNum int) (string, *bytes.Buffer) {
	bounds := screenshot.GetDisplayBounds(disNum)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
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
	fileName := fmt.Sprintf("%d_%dx%d.png", disNum, bounds.Dx(), bounds.Dy())
	var buf bytes.Buffer
	png.Encode(&buf, img)
	fmt.Printf("#%d : %v \"%s\"\n", disNum, bounds, fileName)
	return fileName, &buf
}

func listenWeb(app *application) {
	// Обработчик для главной страницы
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		displayNumberStr := r.URL.Query().Get("d")
		disNum, _ := strconv.Atoi(displayNumberStr)
		_, buf := screen(disNum)
		// Устанавливаем заголовки ответа
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", strconv.Itoa(len(buf.Bytes())))

		// Отправляем изображение в ответе
		if _, err := w.Write(buf.Bytes()); err != nil {
			log.Printf("Failed to write image to response: %v", err)
		}
	})

	// Запуск веб-сервера на порту
	fmt.Printf("Starting web server on %s", app.config.webPort)
	if err := http.ListenAndServe(":"+app.config.webPort, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
