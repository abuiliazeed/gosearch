package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheStore handles query result caching using Redis.
type CacheStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewCacheStore creates a new CacheStore with the given Redis client and TTL.
func NewCacheStore(addr, password string, db int, ttl time.Duration) (*CacheStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &CacheStore{
		client: client,
		ttl:    ttl,
	}, nil
}

// Get retrieves a cached entry by query key.
func (cs *CacheStore) Get(ctx context.Context, key string) (*CacheEntry, error) {
	data, err := cs.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get cache entry: %w", err)
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache entry: %w", err)
	}

	// Check if expired (in case Redis TTL failed)
	if time.Now().After(entry.ExpiresAt) {
		cs.client.Del(ctx, key)
		return nil, nil
	}

	return &entry, nil
}

// Set stores a cache entry with the configured TTL.
func (cs *CacheStore) Set(ctx context.Context, key string, results interface{}) error {
	entry := CacheEntry{
		Query:     key,
		Results:   results,
		ExpiresAt: time.Now().Add(cs.ttl),
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	if err := cs.client.Set(ctx, key, data, cs.ttl).Err(); err != nil {
		return fmt.Errorf("failed to set cache entry: %w", err)
	}

	return nil
}

// Delete removes a cache entry.
func (cs *CacheStore) Delete(ctx context.Context, key string) error {
	if err := cs.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete cache entry: %w", err)
	}
	return nil
}

// Clear removes all cache entries with the gosearch prefix.
func (cs *CacheStore) Clear(ctx context.Context) error {
	iter := cs.client.Scan(ctx, 0, "gosearch:*", 0).Iterator()
	for iter.Next(ctx) {
		if err := cs.client.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("failed to delete key %s: %w", iter.Val(), err)
		}
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan keys: %w", err)
	}
	return nil
}

// SetTTL updates the TTL for the cache store.
func (cs *CacheStore) SetTTL(ttl time.Duration) {
	cs.ttl = ttl
}

// GetTTL returns the current TTL.
func (cs *CacheStore) GetTTL() time.Duration {
	return cs.ttl
}

// Stats returns cache statistics.
func (cs *CacheStore) Stats(ctx context.Context) (map[string]interface{}, error) {
	info, err := cs.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis stats: %w", err)
	}

	// Get key count with gosearch prefix
	var keyCount int64
	iter := cs.client.Scan(ctx, 0, "gosearch:*", 0).Iterator()
	for iter.Next(ctx) {
		keyCount++
	}

	// Parse memory info
	memory := "Unknown"
	if infoCmd := cs.client.Info(ctx, "memory"); infoCmd.Err() == nil {
		infoStr := infoCmd.Val()
		// Simple parsing for used_memory_human
		// In a real implementation, we might want a more robust parser
		for _, line := range strings.Split(infoStr, "\r\n") {
			if strings.HasPrefix(line, "used_memory_human:") {
				memory = strings.TrimPrefix(line, "used_memory_human:")
				break
			}
		}
	}

	stats := map[string]interface{}{
		"key_count":    keyCount,
		"ttl":          cs.ttl.String(),
		"info":         info,
		"memory_usage": memory,
	}

	return stats, nil
}

// Close closes the Redis client connection.
func (cs *CacheStore) Close() error {
	return cs.client.Close()
}

// Ping checks if the Redis server is responsive.
func (cs *CacheStore) Ping(ctx context.Context) error {
	return cs.client.Ping(ctx).Err()
}

// GenerateKey creates a cache key from a query string.
func GenerateKey(query string) string {
	return fmt.Sprintf("gosearch:query:%s", query)
}
