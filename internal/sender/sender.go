package sender

import (
	"delayedNotifier/internal/model"
)

type Sender interface {
	Send(notification *model.Notification) error
}
