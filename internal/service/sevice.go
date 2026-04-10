package service

import (
	"context"
	"time"

	"delayedNotifier/internal/cache"
	"delayedNotifier/internal/model"
	"delayedNotifier/internal/queue"
	"delayedNotifier/internal/repository"
)

type Service struct {
	repo  *repository.Repository
	queue *queue.Queue
	cache *cache.Cache
}

func New(repo *repository.Repository, queue *queue.Queue, cache *cache.Cache) *Service {
	return &Service{
		repo:  repo,
		queue: queue,
		cache: cache,
	}
}

func (s *Service) CreateNotification(ctx context.Context, notification *model.Notification) error {
	if notification.SendAt.Before(time.Now()) {
		return model.ErrInvalidSendAt
	}

	if notification.Channel == "" {
		return model.ErrInvalidChannel
	}

	if notification.Recipient == "" {
		return model.ErrInvalidRecipient
	}

	if notification.Message == "" {
		return model.ErrInvalidMessage
	}

	notification.Status = model.StatusPending
	notification.Attempts = 0

	if notification.MaxAttempts == 0 {
		notification.MaxAttempts = 5
	}
	if err := s.repo.CreateNotification(notification); err != nil {
		return err
	}

	_ = s.cache.SetNotification(ctx, notification)

	delay := time.Until(notification.SendAt)
	return s.queue.PublishNotification(ctx, notification.ID, delay)
}

func (s *Service) GetNotificationByID(id int64) (*model.Notification, error) {
	ctx := context.Background()

	notification, err := s.cache.GetNotification(ctx, id)
	if err == nil {
		return notification, nil
	}

	notification, err = s.repo.GetNotificationByID(id)
	if err != nil {
		return nil, err
	}

	_ = s.cache.SetNotification(ctx, notification)
	return notification, nil
}

func (s *Service) CancelNotificationByID(id int64) error {
	if err := s.repo.CancelNotificationByID(id); err != nil {
		return err
	}

	ctx := context.Background()

	notification, err := s.repo.GetNotificationByID(id)
	if err == nil {
		_ = s.cache.SetNotification(ctx, notification)
	}

	return nil
}

func (s *Service) ListNotifications() ([]model.Notification, error) {
	return s.repo.ListNotifications()
}
