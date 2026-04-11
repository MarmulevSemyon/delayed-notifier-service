package sender

import (
	"fmt"
	"strconv"

	"delayedNotifier/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type TelegramSender struct {
	bot *tgbotapi.BotAPI
}

func NewTelegramSender(token string) (*TelegramSender, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("create telegram bot api: %w", err)
	}

	return &TelegramSender{bot: bot}, nil
}

func (s *TelegramSender) Send(notification *model.Notification) error {
	chatID, err := strconv.ParseInt(notification.Recipient, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid telegram chat id: %w", err)
	}

	msg := tgbotapi.NewMessage(chatID, notification.Message)

	if _, err := s.bot.Send(msg); err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}

	return nil
}
