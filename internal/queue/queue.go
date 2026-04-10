package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/retry"
)

const (
	ExchangeName   = "notifications.exchange"
	ReadyQueueName = "notifications.ready"
	DelayQueueName = "notifications.delay"

	ReadyRoutingKey = "notifications.ready"
	DelayRoutingKey = "notifications.delay"
)

type NotificationMessage struct {
	NotificationID int64 `json:"notification_id"`
}

type Queue struct {
	client    *rabbitmq.RabbitClient
	publisher *rabbitmq.Publisher
}

func New(url string) (*Queue, error) {

	client, err := rabbitmq.NewClient(rabbitmq.ClientConfig{
		URL:            url,
		ConnectionName: "delayed-notifier",
		ConnectTimeout: 5 * time.Second,
		Heartbeat:      10 * time.Second,
		ReconnectStrat: retry.Strategy{Attempts: 3, Delay: time.Second, Backoff: 2},
		ProducingStrat: retry.Strategy{Attempts: 3, Delay: time.Second, Backoff: 2},
		ConsumingStrat: retry.Strategy{Attempts: 3, Delay: time.Second, Backoff: 2},
	})
	if err != nil {
		return nil, fmt.Errorf("create rabbit client: %w", err)
	}

	q := &Queue{
		client:    client,
		publisher: rabbitmq.NewPublisher(client, ExchangeName, "application/json"),
	}

	if err := q.declareTopology(); err != nil {
		return nil, err
	}

	return q, nil
}

func (q *Queue) Close() error {
	return q.client.Close()
}

func (q *Queue) declareTopology() error {
	if err := q.client.DeclareExchange(
		ExchangeName,
		"direct",
		true,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}

	if err := q.client.DeclareQueue(
		ReadyQueueName,
		ExchangeName,
		ReadyRoutingKey,
		true,
		false,
		true,
		nil,
	); err != nil {
		return fmt.Errorf("declare ready queue: %w", err)
	}

	delayArgs := amqp091.Table{
		"x-dead-letter-exchange":    ExchangeName,
		"x-dead-letter-routing-key": ReadyRoutingKey,
	}

	if err := q.client.DeclareQueue(
		DelayQueueName,
		ExchangeName,
		DelayRoutingKey,
		true,
		false,
		true,
		delayArgs,
	); err != nil {
		return fmt.Errorf("declare delay queue: %w", err)
	}

	return nil
}

func (q *Queue) PublishNotification(ctx context.Context, notificationID int64, delay time.Duration) error {
	msg := NotificationMessage{
		NotificationID: notificationID,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal queue message: %w", err)
	}

	if delay < 0 {
		delay = 0
	}

	return q.publisher.Publish(
		ctx,
		body,
		DelayRoutingKey,
		rabbitmq.WithExpiration(delay),
	)
}
