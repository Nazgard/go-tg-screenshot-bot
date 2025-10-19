package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image/png"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/umputun/go-flags"
	"golang.org/x/net/proxy"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kbinani/screenshot"
)

var lastDisplay = 0

// Структура application содержит конфигурацию и Telegram-бота
type application struct {
	Config *Config
	tgBot  *tgbotapi.BotAPI
}

type Config struct {
	Token         string      `long:"telegram-bot-token" env:"TOKEN" description:"Telegram bot token"`
	Debug         bool        `long:"debug" env:"DEBUG" description:"Telegram debug mode"`
	WebPort       string      `long:"addr" env:"ADDR" default:":8080" description:"Web server address"`
	Daily         bool        `long:"daily" env:"DAILY" description:"Enable daily screenshot at 00:30"`
	AllowedChatID int64       `long:"allowed-chat-id" env:"ALLOWED_CHAT_ID" required:"true" description:"Allowed Telegram chat ID"`
	Proxy         ProxyConfig `group:"Proxy" env-namespace:"PROXY"`
}

type ProxyConfig struct {
	Enable         bool   `long:"proxy-enable" env:"ENABLE" description:"Proxy toggle"`
	Socks5Addr     string `long:"proxy-socks5-addr" env:"ADDR" description:"Socks5 proxy address"`
	Socks5User     string `long:"proxy-socks5-user" env:"USER" description:"Socks5 proxy username"`
	Socks5Password string `long:"proxy-socks5-password" env:"PASSWORD" description:"Socks5 proxy password"`
}

func main() {
	// Инициализация логирования в консоль
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	app := configure()

	// Запуск горутин
	go listenTg(app)
	go listenWeb(app)
	go listenDaily(app)

	select {} // Блокируем main, чтобы горутины работали
}

// Configure настраивает флаги и инициализирует приложение
func configure() *application {
	config := &Config{}
	if _, err := flags.Parse(config); err != nil {
		log.Fatal(err)
	}
	flag.Parse()

	httpClient := configureHttpClient(config)

	// Инициализация Telegram-бота
	bot, err := tgbotapi.NewBotAPIWithClient(config.Token, tgbotapi.APIEndpoint, &httpClient)
	if err != nil {
		log.Panicf("Error creating bot: %v", err)
	}
	bot.Debug = config.Debug
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Возвращаем приложение с конфигурацией
	return &application{
		Config: config,
		tgBot:  bot,
	}
}

func configureHttpClient(c *Config) http.Client {
	if c.Proxy.Enable {
		var auth proxy.Auth
		if c.Proxy.Socks5User != "" && c.Proxy.Socks5Password != "" {
			auth = proxy.Auth{
				User:     c.Proxy.Socks5User,
				Password: c.Proxy.Socks5Password,
			}
		}

		dealer, err := proxy.SOCKS5("tcp", c.Proxy.Socks5Addr, &auth, proxy.Direct)
		if err != nil {
			log.Printf("Can't connect to the proxy: %s", err.Error())
		}

		dealContext := func(ctx context.Context, network, address string) (net.Conn, error) {
			return dealer.Dial(network, address)
		}

		tr := &http.Transport{DialContext: dealContext}

		return http.Client{Transport: tr}
	}

	return http.Client{}
}

// isAllowedChatID проверяет, разрешён ли этот chat_id
func isAllowedChatID(app *application, chatID int64) bool {
	return chatID == app.Config.AllowedChatID
}

// listenTg обрабатывает входящие сообщения Telegram
func listenTg(app *application) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := app.tgBot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		text := update.Message.Text

		// Логируем входящее сообщение
		log.Printf("Received message from chat ID %d: %s", chatID, text)

		// Обработка команды /whoami
		if text == "/whoami" {
			reply := tgbotapi.NewMessage(chatID, fmt.Sprintf("Your chat ID: %d", chatID))
			_, err := app.tgBot.Send(reply)
			if err != nil {
				log.Printf("Failed to send /whoami reply: %v", err)
			}
			continue
		}

		// Проверка разрешения на доступ
		if !isAllowedChatID(app, chatID) {
			log.Printf("Unauthorized access attempt from chat ID %d", chatID)
			continue
		}

		// Обработка запросов на скриншоты
		go func() {
			disNum, _ := strconv.Atoi(text)
			lastDisplay = disNum
			fileName, buf := screen(disNum)
			msg := buildPhotoMessage(chatID, fileName, buf)
			_, err := app.tgBot.Send(&msg)
			if err != nil {
				log.Printf("Failed to send screenshot: %v", err)
			}
		}()
	}
}

// listenWeb запускает HTTP-сервер и обрабатывает запросы на получение скриншотов
func listenWeb(app *application) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем IP-адрес клиента из заголовка RemoteAddr
		ipAddress := r.RemoteAddr
		// Извлекаем IP-адрес без порта, если он есть
		host, _, err := net.SplitHostPort(ipAddress)
		if err != nil {
			// Если не удалось разделить IP и порт, используем полное значение
			host = ipAddress
		}

		// Логируем IP-адрес клиента, который пытается получить скриншот
		log.Printf("Received screenshot request from IP address: %s", host)

		// Получение параметра "d" из запроса (номер дисплея)
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

	// Логируем запуск веб-сервера
	log.Printf("Starting web server on port %s", app.Config.WebPort)
	if err := http.ListenAndServe(app.Config.WebPort, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// listenDaily отправляет скриншот каждый день в 00:30, если включено
func listenDaily(app *application) {
	if !app.Config.Daily {
		// Логирование, если опция ежедневного скриншота выключена
		log.Println("Daily screenshot feature is disabled in the config.")
		return
	}

	// Логирование, если опция ежедневного скриншота включена
	log.Println("Daily screenshot feature is enabled. The bot will send a screenshot every day at 00:30.")

	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 0, 30, 0, 0, now.Location())
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
		duration := time.Until(next)

		// Логирование времени до следующего скриншота
		log.Printf("Next daily screenshot will be sent in %v (at 00:30).", duration)

		timer := time.NewTimer(duration)
		<-timer.C

		// Логирование отправки скриншота
		log.Printf("Sending daily screenshot for chat ID %d", app.Config.AllowedChatID)

		fileName, buf := screen(lastDisplay)
		msg := buildPhotoMessage(app.Config.AllowedChatID, fileName, buf)
		_, err := app.tgBot.Send(&msg)
		if err != nil {
			log.Printf("Failed to send daily screenshot: %v", err)
		} else {
			log.Printf("Successfully sent daily screenshot to chat ID %d", app.Config.AllowedChatID)
		}
	}
}

// buildPhotoMessage создаёт сообщение с изображением
func buildPhotoMessage(chatID int64, fileName string, buf *bytes.Buffer) tgbotapi.PhotoConfig {
	return tgbotapi.PhotoConfig{
		BaseFile: tgbotapi.BaseFile{
			BaseChat: tgbotapi.BaseChat{
				ChatID: chatID,
			},
			File: tgbotapi.FileReader{
				Name:   fileName,
				Reader: buf,
			},
		},
	}
}

// screen делает скриншот указанного дисплея
func screen(disNum int) (string, *bytes.Buffer) {
	bounds := screenshot.GetDisplayBounds(disNum)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		// Логирование ошибки захвата экрана и попытки повторить
		log.Printf("Error capturing screen #%d: %v. Retrying...", disNum, err)
		for i := 0; i < 5; i++ {
			time.Sleep(1 * time.Second)
			bounds = screenshot.GetDisplayBounds(disNum)
			img, err = screenshot.CaptureRect(bounds)
			if err == nil {
				log.Printf("Retry successful for screen #%d", disNum)
				break
			}
		}
		if err != nil {
			log.Printf("Capture failed for screen #%d: %v", disNum, err)
			return "", nil
		}
	}

	// Логирование успешного захвата экрана
	fileName := fmt.Sprintf("%d_%dx%d.png", disNum, bounds.Dx(), bounds.Dy())
	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	if err != nil {
		log.Printf("Encode failed for screen #%d: %v", disNum, err)
		return "", nil
	}
	log.Printf("Captured screen #%d: %s", disNum, fileName)
	return fileName, &buf
}
