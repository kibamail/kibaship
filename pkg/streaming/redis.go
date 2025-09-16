/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package streaming

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisClient implements RedisClient interface using go-redis
type redisClient struct {
	client *redis.Client
}

// NewRedisClient creates a new Redis client for Valkey/Redis
func NewRedisClient(address, password string) RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr:         address,
		Password:     password,
		DB:           0, // Default database
		DialTimeout:  connectionTimeout,
		ReadTimeout:  operationTimeout,
		WriteTimeout: operationTimeout,
		PoolSize:     poolSize,
		MinIdleConns: minIdleConns,
		MaxRetries:   maxRetries,
	})

	return &redisClient{
		client: rdb,
	}
}

// XAdd adds an entry to a Redis stream
func (r *redisClient) XAdd(ctx context.Context, stream string, values map[string]interface{}) (string, error) {
	if stream == "" {
		return "", fmt.Errorf("stream name cannot be empty")
	}
	if len(values) == 0 {
		return "", fmt.Errorf("values cannot be empty")
	}

	result := r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	})

	return result.Result()
}

// Ping tests the connection to Redis/Valkey
func (r *redisClient) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Close closes the Redis client connection
func (r *redisClient) Close() error {
	return r.client.Close()
}

// Redis client configuration constants
const (
	connectionTimeout = 30 * time.Second // Connection timeout
	operationTimeout  = 10 * time.Second // Read/write timeout
	poolSize          = 100              // Connection pool size
	minIdleConns      = 10               // Minimum idle connections
	maxRetries        = 3                // Maximum retry attempts
)
