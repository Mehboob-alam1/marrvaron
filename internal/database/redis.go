package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"marvaron/internal/config"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client
var Ctx = context.Background()

func ConnectRedis() error {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     config.AppConfig.GetRedisAddr(),
		Password: config.AppConfig.Redis.Password,
		DB:       config.AppConfig.Redis.DB,
	})

	// Test connection
	_, err := RedisClient.Ping(Ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("Redis connected successfully")
	return nil
}

func CloseRedis() error {
	return RedisClient.Close()
}

// Cache helpers
func SetCache(key string, value interface{}, expiration time.Duration) error {
	return RedisClient.Set(Ctx, key, value, expiration).Err()
}

func GetCache(key string) (string, error) {
	return RedisClient.Get(Ctx, key).Result()
}

func DeleteCache(key string) error {
	return RedisClient.Del(Ctx, key).Err()
}

func ExistsCache(key string) (bool, error) {
	count, err := RedisClient.Exists(Ctx, key).Result()
	return count > 0, err
}
