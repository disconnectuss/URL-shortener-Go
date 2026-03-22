package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
	ttl    time.Duration
}

func New(addr string) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Cache{
		client: client,
		ttl:    10 * time.Minute,
	}, nil
}

func (c *Cache) Get(shortCode string) (string, error) {
	ctx := context.Background()
	return c.client.Get(ctx, "url:"+shortCode).Result()
}

func (c *Cache) Set(shortCode, originalURL string) error {
	ctx := context.Background()
	return c.client.Set(ctx, "url:"+shortCode, originalURL, c.ttl).Err()
}

func (c *Cache) Close() error {
	return c.client.Close()
}
