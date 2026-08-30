package sender

import (
	"github.com/MarmulevSemyon/delayed-notifier-service/internal/model"
)

type Sender interface {
	Send(notification *model.Notification) error
}
