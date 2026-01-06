package main

import (
	"bytes"
	"context"
	"crypto/subtle"
	"flag"
	"fmt"
	"image/png"
	"io"
	"log"
	"net"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/umputun/go-flags"
	"golang.org/x/net/proxy"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kbinani/screenshot"
)

// ====================== ИНТЕРФЕЙСЫ ДЛЯ ТЕСТИРУЕМОСТИ ======================

type ScreenshotCapturer interface {
	CaptureDisplay(display int) (filename string, imageData io.Reader, err error)
}

type MessageSender interface {
	SendPhoto(chatID int64, filename string, photo io.Reader) error
}

type Logger interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
	Fatalf(format string, v ...interface{})
}

// ====================== РЕАЛЬНЫЕ РЕАЛИЗАЦИИ ======================

type realScreenshotCapturer struct{}

func (r *realScreenshotCapturer) CaptureDisplay(display int) (string, io.Reader, error) {
	bounds := screenshot.GetDisplayBounds(display)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		for i := 0; i < 5; i++ {
			time.Sleep(1 * time.Second)
			bounds = screenshot.GetDisplayBounds(display)
			img, err = screenshot.CaptureRect(bounds)
			if err == nil {
				break
			}
		}
		if err != nil {
			return "", nil, fmt.Errorf("capture failed for display %d after retries: %w", display, err)
		}
	}

	filename := fmt.Sprintf("%d_%dx%d.png", display, bounds.Dx(), bounds.Dy())

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", nil, fmt.Errorf("png encode failed for display %d: %w", display, err)
	}

	return filename, &buf, nil
}

type telegramMessageSender struct {
	bot *tgbotapi.BotAPI
}

func (t *telegramMessageSender) SendPhoto(chatID int64, filename string, photo io.Reader) error {
	msg := tgbotapi.NewPhoto(chatID, tgbotapi.FileReader{
		Name:   filename,
		Reader: photo,
	})
	_, err := t.bot.Send(msg)
	return err
}

type stdLogger struct{}

func (s stdLogger) Printf(format string, v ...interface{}) { log.Printf(format, v...) }
func (s stdLogger) Println(v ...interface{})               { log.Println(v...) }
func (s stdLogger) Fatalf(format string, v ...interface{}) { log.Fatalf(format, v...) }

// ====================== ОСНОВНЫЕ СТРУКТУРЫ ======================

type Application struct {
	Config             *Config
	TgBot              *tgbotapi.BotAPI
	ScreenshotCapturer ScreenshotCapturer
	MessageSender      MessageSender
	Logger             Logger
	LastDisplay        int
}

type Config struct {
	Token         string      `long:"telegram-bot-token" env:"TOKEN" description:"Telegram bot token"`
	Debug         bool        `long:"debug" env:"DEBUG" description:"Telegram debug mode"`
	WebPort       string      `long:"addr" env:"ADDR" default:":8080" description:"Web server address"`
	Daily         bool        `long:"daily" env:"DAILY" description:"Enable daily screenshot at 00:30"`
	AllowedChatID int64       `long:"allowed-chat-id" env:"ALLOWED_CHAT_ID" required:"true" description:"Allowed Telegram chat ID"`
	Proxy         ProxyConfig `group:"Proxy" env-namespace:"PROXY"`
	AuthConfig    AuthConfig  `group:"Auth" env-namespace:"AUTH"`
}

type ProxyConfig struct {
	Enable         bool   `long:"proxy-enable" env:"ENABLE" description:"Proxy toggle"`
	Socks5Addr     string `long:"proxy-socks5-addr" env:"ADDR" description:"Socks5 proxy address"`
	Socks5User     string `long:"proxy-socks5-user" env:"USER" description:"Socks5 proxy username"`
	Socks5Password string `long:"proxy-socks5-password" env:"PASSWORD" description:"Socks5 proxy password"`
}

type AuthConfig struct {
	Enabled bool   `long:"auth-enable" env:"ENABLE" description:"Enable auth"`
	User    string `long:"auth-user" env:"USER" description:"User name"`
	Pass    string `long:"auth-password" env:"PASS" description:"Password name"`
}

// ====================== ЗАПУСК С GRACEFUL SHUTDOWN ======================

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	app := Configure()

	// Создаём контекст с отменой для graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Канал для уведомления о завершении всех горутин
	shutdownComplete := make(chan struct{})

	// Запускаем компоненты
	go app.ListenTelegram(ctx)
	go app.ListenWeb(ctx)
	go app.ListenDaily(ctx)

	// Ждём сигнала завершения
	<-ctx.Done()
	app.Logger.Println("Shutdown signal received. Starting graceful shutdown...")

	// Даём время на завершение текущих операций (например, отправка скриншота)
	shutdownTimeout := 30 * time.Second
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Ждём завершения горутин или таймаута
	select {
	case <-shutdownComplete:
		app.Logger.Println("All components stopped gracefully.")
	case <-shutdownCtx.Done():
		app.Logger.Println("Shutdown timeout exceeded. Forcing exit.")
	}

	app.Logger.Println("Application stopped.")
}

// Configure — инициализация приложения
func Configure() *Application {
	config := &Config{}
	if _, err := flags.Parse(config); err != nil {
		log.Fatal(err)
	}
	flag.Parse()

	httpClient := configureHttpClient(config)

	bot, err := tgbotapi.NewBotAPIWithClient(config.Token, tgbotapi.APIEndpoint, &httpClient)
	if err != nil {
		log.Panicf("Error creating bot: %v", err)
	}
	bot.Debug = config.Debug
	log.Printf("Authorized on account %s", bot.Self.UserName)

	return &Application{
		Config:             config,
		TgBot:              bot,
		ScreenshotCapturer: &realScreenshotCapturer{},
		MessageSender:      &telegramMessageSender{bot: bot},
		Logger:             stdLogger{},
		LastDisplay:        0,
	}
}

func configureHttpClient(c *Config) http.Client {
	if !c.Proxy.Enable {
		return http.Client{}
	}

	var auth *proxy.Auth
	if c.Proxy.Socks5User != "" && c.Proxy.Socks5Password != "" {
		auth = &proxy.Auth{
			User:     c.Proxy.Socks5User,
			Password: c.Proxy.Socks5Password,
		}
	}

	dialer, err := proxy.SOCKS5("tcp", c.Proxy.Socks5Addr, auth, proxy.Direct)
	if err != nil {
		log.Printf("Can't connect to the proxy: %s (continuing without proxy)", err)
		return http.Client{}
	}

	return http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
		},
	}
}

// ====================== БИЗНЕС-ЛОГИКА ======================

func (app *Application) IsAllowedChatID(chatID int64) bool {
	return chatID == app.Config.AllowedChatID
}

func (app *Application) HandleTelegramMessage(text string, chatID int64) error {
	if text == "/whoami" {
		return app.MessageSender.SendPhoto(chatID, "whoami.png", bytes.NewBufferString(fmt.Sprintf("Your chat ID: %d", chatID)))
	}

	if !app.IsAllowedChatID(chatID) {
		app.Logger.Printf("Unauthorized access attempt from chat ID %d", chatID)
		return fmt.Errorf("unauthorized chat ID: %d", chatID)
	}

	disNum, err := strconv.Atoi(text)
	if err != nil {
		return fmt.Errorf("invalid display number: %s", text)
	}

	app.LastDisplay = disNum

	filename, photo, err := app.ScreenshotCapturer.CaptureDisplay(disNum)
	if err != nil {
		return fmt.Errorf("failed to capture screenshot: %w", err)
	}

	return app.MessageSender.SendPhoto(chatID, filename, photo)
}

// ====================== СЛУШАТЕЛИ С ПОДДЕРЖКОЙ КОНТЕКСТА ======================

// ListenTelegram — обработка обновлений Telegram с поддержкой отмены по контексту
func (app *Application) ListenTelegram(parentCtx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := app.TgBot.GetUpdatesChan(u)

	for {
		select {
		case <-parentCtx.Done():
			app.Logger.Println("Stopping Telegram listener...")
			app.TgBot.StopReceivingUpdates() // Важно: явно останавливаем получение обновлений
			return
		case update, ok := <-updates:
			if !ok {
				app.Logger.Println("Telegram updates channel closed.")
				return
			}
			if update.Message == nil {
				continue
			}

			chatID := update.Message.Chat.ID
			text := update.Message.Text

			app.Logger.Printf("Received message from chat ID %d: %s", chatID, text)

			go func(text string, chatID int64) {
				if err := app.HandleTelegramMessage(text, chatID); err != nil {
					app.Logger.Printf("Error handling message: %v", err)
				}
			}(text, chatID)
		}
	}
}

// ListenWeb — HTTP-сервер с graceful shutdown
func (app *Application) ListenWeb(parentCtx context.Context) {
	srv := &http.Server{
		Addr: app.Config.WebPort,
	}

	basicAuth := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !app.Config.AuthConfig.Enabled {
				next(w, r)
				return
			}

			user, pass, ok := r.BasicAuth()
			if !ok ||
				subtle.ConstantTimeCompare([]byte(user), []byte(app.Config.AuthConfig.User)) != 1 ||
				subtle.ConstantTimeCompare([]byte(pass), []byte(app.Config.AuthConfig.Pass)) != 1 {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted Area", charset="UTF-8"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next(w, r)
		}
	}

	http.HandleFunc("/", basicAuth(func(w http.ResponseWriter, r *http.Request) {
		// Проверка контекста — если уже shutdown, не обрабатывать новые запросы
		if r.Context().Err() != nil {
			http.Error(w, "Server is shutting down", http.StatusServiceUnavailable)
			return
		}

		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		if host == "" {
			host = r.RemoteAddr
		}
		app.Logger.Printf("Received screenshot request from IP: %s", host)

		displayStr := r.URL.Query().Get("d")
		disNum, err := strconv.Atoi(displayStr)
		if err != nil || disNum < 0 {
			http.Error(w, "Invalid display number", http.StatusBadRequest)
			return
		}

		_, imgReader, err := app.ScreenshotCapturer.CaptureDisplay(disNum)
		if err != nil {
			app.Logger.Printf("Capture failed: %v", err)
			http.Error(w, "Failed to capture screenshot", http.StatusInternalServerError)
			return
		}

		data, _ := io.ReadAll(imgReader)
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		_, _ = w.Write(data)
	}))

	go func() {
		app.Logger.Printf("Starting web server on %s", app.Config.WebPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.Logger.Fatalf("Web server error: %v", err)
		}
	}()

	// Ждём сигнала завершения
	<-parentCtx.Done()
	app.Logger.Println("Stopping web server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		app.Logger.Printf("Web server forced to shutdown: %v", err)
	} else {
		app.Logger.Println("Web server stopped gracefully.")
	}
}

// ListenDaily — ежедневная отправка с проверкой контекста
func (app *Application) ListenDaily(parentCtx context.Context) {
	if !app.Config.Daily {
		app.Logger.Println("Daily screenshot feature is disabled.")
		return
	}

	app.Logger.Println("Daily screenshot enabled — sending at 00:30 every day.")

	for {
		select {
		case <-parentCtx.Done():
			app.Logger.Println("Stopping daily screenshot scheduler...")
			return
		default:
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), 0, 30, 0, 0, now.Location())
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}

			duration := time.Until(next)
			app.Logger.Printf("Next daily screenshot in %v (at %s)", duration, next.Format("2006-01-02 15:04"))

			// Спим с проверкой контекста
			select {
			case <-parentCtx.Done():
				app.Logger.Println("Daily scheduler interrupted during sleep.")
				return
			case <-time.After(duration):
				// Время пришло — отправляем
			}

			app.Logger.Printf("Sending daily screenshot (display %d) to chat ID %d", app.LastDisplay, app.Config.AllowedChatID)

			filename, photo, err := app.ScreenshotCapturer.CaptureDisplay(app.LastDisplay)
			if err != nil {
				app.Logger.Printf("Daily capture failed: %v", err)
				continue
			}

			if err := app.MessageSender.SendPhoto(app.Config.AllowedChatID, filename, photo); err != nil {
				app.Logger.Printf("Failed to send daily screenshot: %v", err)
			} else {
				app.Logger.Printf("Daily screenshot sent successfully")
			}
		}
	}
}
