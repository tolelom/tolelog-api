package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache wraps a Redis client for simple get/set/delete operations with JSON serialization.
type Cache struct {
	client *redis.Client
}

// New creates a new Cache instance connected to the given Redis address.
// Returns an error if the connection cannot be established within 3 seconds.
func New(addr string) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &Cache{client: client}, nil
}

// Get retrieves a cached value by key and unmarshals it into dest.
func (c *Cache) Get(key string, dest interface{}) error {
	ctx := context.Background()
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

// Set serializes value as JSON and stores it in Redis with the given TTL.
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) error {
	ctx := context.Background()
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

// Delete removes one or more keys from Redis.
func (c *Cache) Delete(keys ...string) error {
	ctx := context.Background()
	return c.client.Del(ctx, keys...).Err()
}

// DeleteByPattern scans for keys matching the given glob pattern and deletes them.
func (c *Cache) DeleteByPattern(pattern string) error {
	ctx := context.Background()
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(keys) > 0 {
		return c.client.Unlink(ctx, keys...).Err()
	}
	return nil
}

// Ping verifies the Redis connection within the given timeout.
func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Close releases the underlying Redis client.
func (c *Cache) Close() error {
	return c.client.Close()
}
