package main

import (
	"fmt"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kbinani/screenshot"
)

const (
	botToken = ""
	watchDir = "./screenshots"
)

var myChatID int64 = 1689350885

func main() {
	_ = os.MkdirAll(watchDir, os.ModePerm)

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic("Ошибка инициализации бота:", err)
	}

	log.Printf("Бот успешно запущен под именем %s", bot.Self.UserName)

	go watchFolder(bot)
	go startScreenshotLoop()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] ID чата: %d | Текст: %s", update.Message.From.UserName, update.Message.Chat.ID, update.Message.Text)

		if myChatID == 0 {
			myChatID = update.Message.Chat.ID
			msg := tgbotapi.NewMessage(myChatID, "Вы привязали этот чат! Теперь сюда будут приходить скриншоты.")
			bot.Send(msg)
		}
	}
}

func watchFolder(bot *tgbotapi.BotAPI) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Ошибка создания вотчера:", err)
	}
	defer watcher.Close()

	err = watcher.Add(watchDir)
	if err != nil {
		log.Fatal("Ошибка добавления папки для слежения:", err)
	}

	log.Printf("Слежу за папкой: %s", watchDir)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Create) {
				ext := strings.ToLower(filepath.Ext(event.Name))
				if ext == ".png" || ext == ".jpg" || ext == ".jpeg" {
					log.Printf("Обнаружен новый скриншот: %s", event.Name)

					time.Sleep(300 * time.Millisecond)

					if myChatID != 0 {
						sendScreenshot(bot, myChatID, event.Name)
					} else {
						log.Println("Предупреждение: Скриншот есть, но ID чата неизвестен!")
					}
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("Ошибка вотчера:", err)
		}
	}
}

func sendScreenshot(bot *tgbotapi.BotAPI, chatID int64, filePath string) {
	photoBytes, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Не удалось прочитать файл %s: %v", filePath, err)
		return
	}

	file := tgbotapi.FileBytes{
		Name:  filepath.Base(filePath),
		Bytes: photoBytes,
	}

	photoMsg := tgbotapi.NewPhoto(chatID, file)
	photoMsg.Caption = "Новый скриншот: " + filepath.Base(filePath)

	_, err = bot.Send(photoMsg)
	if err != nil {
		log.Printf("Ошибка отправки фото в Telegram: %v", err)
	} else {
		log.Println("Скриншот успешно отправлен!")

		_ = os.Remove(filePath)
	}
}

func startScreenshotLoop() {
	for {

		image, err := screenshot.CaptureDisplay(0)
		if err != nil {
			log.Printf("Ошибка захвата экрана: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}

		timestamp := time.Now().Format("2006-01-02_15-04-05")
		fileName := fmt.Sprintf("screenshot_%s.png", timestamp)
		fullPath := filepath.Join(watchDir, fileName)

		err = func() error {
			file, err := os.Create(fullPath)
			if err != nil {
				return fmt.Errorf("не удалось создать файл: %w", err)
			}
			defer file.Close()

			if err := png.Encode(file, image); err != nil {
				return fmt.Errorf("не удалось закодировать PNG: %w", err)
			}
			return nil
		}()

		if err != nil {
			log.Println("Ошибка при сохранении:", err)
		}

		time.Sleep(3 * time.Second)
	}
}
