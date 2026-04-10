package model

import "errors"

var (
	ErrInvalidSendAt    = errors.New("send_at must be in the future")
	ErrInvalidChannel   = errors.New("channel is required")
	ErrInvalidRecipient = errors.New("recipient is required")
	ErrInvalidMessage   = errors.New("message is required")
)
