package sender

import (
	"fmt"

	"delayedNotifier/internal/model"
)

type Router struct {
	mock     Sender
	telegram Sender
}

func NewRouter(mock Sender, telegram Sender) *Router {
	return &Router{
		mock:     mock,
		telegram: telegram,
	}
}

func (r *Router) Send(notification *model.Notification) error {
	switch notification.Channel {
	case "mock":
		return r.mock.Send(notification)
	case "telegram":
		if r.telegram == nil {
			return fmt.Errorf("telegram sender is not configured")
		}
		return r.telegram.Send(notification)
	default:
		return fmt.Errorf("unsupported channel: %s", notification.Channel)
	}
}
