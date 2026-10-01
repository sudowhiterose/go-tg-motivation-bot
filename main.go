package main

import (
	"log"
	"os"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var motivations = []string{
	"Дисциплина — это решение делать то, чего очень не хочется делать, чтобы достичь того, чего очень хочется достичь.",
	"Каждый день — это новая возможность стать лучше, чем вчера.",
	"Не бойся идти медленно, бойся стоять на месте.",
	"Успех — это сумма маленьких усилий, повторяющихся изо дня в день.",
}

type MotivationManager struct {
	mu       sync.Mutex
	stopChan chan struct{}
	running  bool
}

func main() {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("Критическая ошибка: переменная TELEGRAM_BOT_TOKEN не задана")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Авторизован под аккаунтом %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	manager := &MotivationManager{}
	quoteIndex := 0

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID

		switch update.Message.Text {
		case "/start_motivation":
			manager.mu.Lock()
			if manager.running {
				msg := tgbotapi.NewMessage(chatID, "Рассылка мотивации уже запущена!")
				bot.Send(msg)
				manager.mu.Unlock()
				continue
			}

			manager.running = true
			manager.stopChan = make(chan struct{})
			manager.mu.Unlock()

			msg := tgbotapi.NewMessage(chatID, "🚀 Мотивация активирована! Буду писать вам каждые 30 минут.")
			bot.Send(msg)

			go func(cID int64, stop <-chan struct{}) {
				ticker := time.NewTicker(30 * time.Minute)
				defer ticker.Stop()

				for {
					select {
					case <-ticker.C:
						text := "💪 Мотивация на этот час:\n\n" + motivations[quoteIndex]
						quoteIndex = (quoteIndex + 1) % len(motivations)

						msg := tgbotapi.NewMessage(cID, text)
						bot.Send(msg)

					case <-stop:
						log.Printf("Фоновый таймер для чата %d остановлен", cID)
						return
					}
				}
			}(chatID, manager.stopChan)

		case "/stop_motivation":
			manager.mu.Lock()
			if !manager.running {
				msg := tgbotapi.NewMessage(chatID, "Рассылка и так выключена.")
				bot.Send(msg)
				manager.mu.Unlock()
				continue
			}

			close(manager.stopChan)
			manager.running = false
			manager.mu.Unlock()

			msg := tgbotapi.NewMessage(chatID, "🛑 Мотивация поставлена на паузу.")
			bot.Send(msg)

		default:
			replyText := "❌ Я не знаю такой команды. Используйте /start_motivation или /stop_motivation"
			msg := tgbotapi.NewMessage(chatID, replyText)
			msg.ReplyToMessageID = update.Message.MessageID
			bot.Send(msg)
		}
	}
}
