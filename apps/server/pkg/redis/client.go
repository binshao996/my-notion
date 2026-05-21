package redis

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

// Connect connects to a Redis instance at the given address.
// It parses the URL, creates a client, and pings to verify connectivity.
func Connect(addr string) (*redis.Client, error) {
	if addr == "" {
		addr = "redis://localhost:6379"
	}

	opts, err := redis.ParseURL(addr)
	if err != nil {
		return nil, fmt.Errorf("redis parse url: %w", err)
	}

	client := redis.NewClient(opts)

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	log.Printf("redis: connected to %s", addr)
	return client, nil
}
