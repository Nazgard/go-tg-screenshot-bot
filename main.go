package main

import (
	"bytes"
	"flag"
	"fmt"
	"image/png"
	"log"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kbinani/screenshot"
)

func main() {
	token := flag.String("token", "Token", "Enter telegram token")
	debug := flag.Bool("debug", false, "Debug mode")
	flag.Parse()
	bot, err := tgbotapi.NewBotAPI(*token)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = *debug
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			return
		}
		disNum, _ := strconv.Atoi(update.Message.Text)
		bounds := screenshot.GetDisplayBounds(disNum)
		img, err := screenshot.CaptureRect(bounds)
		if err != nil {
			fmt.Printf("Error while capture rect %s", err.Error())
			continue
		}
		fileName := fmt.Sprintf("%d_%dx%d.png", disNum, bounds.Dx(), bounds.Dy())
		var buf bytes.Buffer
		png.Encode(&buf, img)
		fmt.Printf("#%d : %v \"%s\"\n", disNum, bounds, fileName)
		msg := tgbotapi.PhotoConfig{
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{
					ChatID: update.Message.From.ID,
				},
				File: tgbotapi.FileReader{
					Name:   fileName,
					Reader: &buf,
				},
			},
		}
		bot.Send(&msg)
	}

	select {}
}
