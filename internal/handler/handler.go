package handler

import (
	"net/http"
	"strconv"
	"time"

	"delayedNotifier/internal/model"
	"delayedNotifier/internal/repository"
	"delayedNotifier/internal/service"

	"github.com/wb-go/wbf/ginext"
)

type Handler struct {
	service *service.Service
}

func New(service *service.Service) *Handler {
	return &Handler{
		service: service,
	}
}

type createNotificationRequest struct {
	Channel     string `json:"channel"`
	Recipient   string `json:"recipient"`
	Message     string `json:"message"`
	SendAt      string `json:"send_at"`
	MaxAttempts int    `json:"max_attempts"`
}

func (h *Handler) CreateNotification(c *ginext.Context) {
	var req createNotificationRequest

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ginext.H{
			"error": "invalid json body",
		})
		return
	}

	sendAt, err := time.Parse(time.RFC3339, req.SendAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, ginext.H{
			"error": "invalid send_at, use RFC3339 format",
		})
		return
	}

	notification := &model.Notification{
		Channel:     req.Channel,
		Recipient:   req.Recipient,
		Message:     req.Message,
		SendAt:      sendAt,
		MaxAttempts: req.MaxAttempts,
	}

	if err := h.service.CreateNotification(c.Request.Context(), notification); err != nil {
		switch err {
		case model.ErrInvalidSendAt,
			model.ErrInvalidChannel,
			model.ErrInvalidRecipient,
			model.ErrInvalidMessage:
			c.JSON(http.StatusBadRequest, ginext.H{
				"error": err.Error(),
			})
			return
		default:
			c.JSON(http.StatusInternalServerError, ginext.H{
				"error": "could not create notification",
			})
			return
		}
	}

	c.JSON(http.StatusCreated, notification)
}

func (h *Handler) GetNotificationByID(c *ginext.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ginext.H{
			"error": "invalid id",
		})
		return
	}

	notification, err := h.service.GetNotificationByID(id)
	if err != nil {
		if err == repository.ErrNoSuchNotification {
			c.JSON(http.StatusNotFound, ginext.H{
				"error": "notification not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ginext.H{
			"error": "could not get notification",
		})
		return
	}

	c.JSON(http.StatusOK, notification)
}

func (h *Handler) CancelNotificationByID(c *ginext.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ginext.H{
			"error": "invalid id",
		})
		return
	}

	err = h.service.CancelNotificationByID(id)
	if err != nil {
		switch err {
		case repository.ErrNoSuchNotification:
			c.JSON(http.StatusNotFound, ginext.H{
				"error": "notification not found",
			})
			return
		case repository.ErrNotificationNotCanceled:
			c.JSON(http.StatusConflict, ginext.H{
				"error": "notification cannot be canceled",
			})
			return
		default:
			c.JSON(http.StatusInternalServerError, ginext.H{
				"error": "could not cancel notification",
			})
			return
		}
	}

	c.JSON(http.StatusOK, ginext.H{
		"status": "canceled",
	})
}

func (h *Handler) ListNotifications(c *ginext.Context) {
	notifications, err := h.service.ListNotifications()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ginext.H{
			"error": "could not list notifications",
		})
		return
	}

	c.JSON(http.StatusOK, notifications)
}
