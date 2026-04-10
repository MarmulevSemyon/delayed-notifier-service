package sender

import (
	"fmt"
	"log"
	"strings"

	"delayedNotifier/internal/model"
)

type Sender struct{}

func New() *Sender {
	return &Sender{}
}

func (s *Sender) Send(notification *model.Notification) error {
	if notification.Channel != "mock" {
		return fmt.Errorf("unsupported channel: %s", notification.Channel)
	}

	// Для теста: если в тексте есть "fail", считаем отправку ошибкой
	if strings.Contains(strings.ToLower(notification.Message), "fail") {
		return fmt.Errorf("mock send error")
	}

	log.Printf(
		"mock notification sent: id=%d recipient=%s message=%s",
		notification.ID,
		notification.Recipient,
		notification.Message,
	)

	return nil
}
