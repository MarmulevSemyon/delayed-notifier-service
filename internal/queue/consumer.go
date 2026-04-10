package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"delayedNotifier/internal/cache"
	"delayedNotifier/internal/model"
	"delayedNotifier/internal/repository"
	"delayedNotifier/internal/sender"

	"github.com/rabbitmq/amqp091-go"
	"github.com/wb-go/wbf/rabbitmq"
)

func retryDelay(attempt int) time.Duration {
	base := 10 * time.Second
	return base * time.Duration(1<<(attempt-1))
}

func strPtr(s string) *string {
	return &s
}

func (q *Queue) StartConsumer(
	ctx context.Context,
	repo *repository.Repository,
	s *sender.Sender,
	cache *cache.Cache,
) error {
	log.Println("consumer started")

	handler := func(ctx context.Context, d amqp091.Delivery) error {
		var msg NotificationMessage

		if err := json.Unmarshal(d.Body, &msg); err != nil {
			return fmt.Errorf("unmarshal message: %w", err)
		}

		notification, err := repo.GetNotificationByID(msg.NotificationID)
		if err != nil {
			if err == repository.ErrNoSuchNotification {
				log.Printf("notification %d not found, skip", msg.NotificationID)
				return nil
			}
			return err
		}

		switch notification.Status {
		case model.StatusCanceled, model.StatusSent, model.StatusFailed:
			log.Printf("notification %d has status %s, skip", notification.ID, notification.Status)
			return nil
		}

		if err := repo.UpdateStatusByID(notification.ID, model.StatusProcessing, nil); err != nil {
			return err
		}
		notification.Status = model.StatusProcessing
		notification.LastError = nil
		_ = cache.SetNotification(ctx, notification)

		if err := s.Send(notification); err != nil {

			attempts, maxAttempts, incErr := repo.IncreaseAttemptsByID(notification.ID, err.Error())
			if incErr != nil {
				return incErr
			}

			if attempts >= maxAttempts {
				if setErr := repo.UpdateStatusByID(notification.ID, model.StatusFailed, strPtr(err.Error())); setErr != nil {
					return setErr
				}

				notification.Status = model.StatusFailed
				notification.LastError = strPtr(err.Error())
				_ = cache.SetNotification(ctx, notification)

				log.Printf("notification %d failed permanently after %d attempts", notification.ID, attempts)
				return nil
			}

			delay := retryDelay(attempts)
			if pubErr := q.PublishNotification(ctx, notification.ID, delay); pubErr != nil {
				return pubErr
			}

			if setErr := repo.UpdateStatusByID(notification.ID, model.StatusPending, strPtr(err.Error())); setErr != nil {
				return setErr
			}

			notification.Status = model.StatusPending
			notification.LastError = strPtr(err.Error())
			notification.Attempts = attempts
			_ = cache.SetNotification(ctx, notification)

			log.Printf(
				"notification %d send failed, retry %d/%d in %s",
				notification.ID,
				attempts,
				maxAttempts,
				delay,
			)

			return nil
		}

		if err := repo.UpdateStatusByID(notification.ID, model.StatusSent, nil); err != nil {
			return err
		}

		notification.Status = model.StatusSent
		notification.LastError = nil
		_ = cache.SetNotification(ctx, notification)

		log.Printf("notification %d is sent", notification.ID)
		return nil
	}

	consumer := rabbitmq.NewConsumer(q.client, rabbitmq.ConsumerConfig{
		Queue:         ReadyQueueName,
		ConsumerTag:   "delayed-notifier-consumer",
		AutoAck:       false,
		Workers:       1,
		PrefetchCount: 1,
	}, handler)

	return consumer.Start(ctx)
}
