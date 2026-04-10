package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"delayedNotifier/internal/model"

	"github.com/wb-go/wbf/redis"
)

const ttl = 10 * time.Minute

type Cache struct {
	client *redis.Client
}

func New(addr, password string, db int) *Cache {
	return &Cache{
		client: redis.New(addr, password, db),
	}
}

func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx)
}

func (c *Cache) key(id int64) string {
	return fmt.Sprintf("notification:%d", id)
}

func (c *Cache) GetNotification(ctx context.Context, id int64) (*model.Notification, error) {
	val, err := c.client.Get(ctx, c.key(id))
	if err != nil {
		return nil, err
	}

	var n model.Notification
	if err := json.Unmarshal([]byte(val), &n); err != nil {
		return nil, err
	}

	return &n, nil
}

func (c *Cache) SetNotification(ctx context.Context, n *model.Notification) error {
	data, err := json.Marshal(n)
	if err != nil {
		return err
	}

	return c.client.SetWithExpiration(ctx, c.key(n.ID), string(data), ttl)
}

func (c *Cache) DeleteNotification(ctx context.Context, id int64) error {
	return c.client.Del(ctx, c.key(id))
}
