package sender

import (
	"fmt"
	"log"
	"strings"

	"delayedNotifier/internal/model"
)

type MockSender struct{}

func NewMockSender() *MockSender {
	return &MockSender{}
}

func (s *MockSender) Send(notification *model.Notification) error {
	if strings.Contains(strings.ToLower(notification.Message), "fail") {
		return fmt.Errorf("mock send error")
	}

	log.Printf("mock notification sent: id=%d recipient=%s message=%s",
		notification.ID, notification.Recipient, notification.Message)

	return nil
}
