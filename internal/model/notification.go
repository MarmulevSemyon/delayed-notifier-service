package model

import "time"

type NotificationStatus string

const (
	StatusPending    NotificationStatus = "pending"
	StatusProcessing NotificationStatus = "processing"
	StatusSent       NotificationStatus = "sent"
	StatusFailed     NotificationStatus = "failed"
	StatusCanceled   NotificationStatus = "canceled"
)

type Notification struct {
	ID          int64              `json:"id"`
	Channel     string             `json:"channel"`
	Recipient   string             `json:"recipient"`
	Message     string             `json:"message"`
	SendAt      time.Time          `json:"send_at"`
	Status      NotificationStatus `json:"status"`
	Attempts    int                `json:"attempts"`
	MaxAttempts int                `json:"max_attempts"`
	LastError   *string            `json:"last_error,omitempty"`
	CanceledAt  *time.Time         `json:"canceled_at,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}
