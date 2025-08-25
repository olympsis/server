package redis

import (
	"context"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type RedisDatabase struct {
	client redis.UniversalClient
	logger *logrus.Logger
}

func NewClient(addr string, username *string, password *string, db int) redis.UniversalClient {
	opts := redis.Options{}
	opts.Addr = addr
	opts.DB = db

	mode := os.Getenv("MODE")

	if username != nil && mode == "PRODUCTION" {
		opts.Username = *username
	}
	if password != nil && mode == "PRODUCTION" {
		opts.Password = *password
	}

	return redis.NewClient(&opts)
}

func NewClusterClient(addrs []string, password string) redis.UniversalClient {
	return redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    addrs,
		Password: password,
	})
}

func New(client *redis.UniversalClient, logger *logrus.Logger) RedisDatabase {
	return RedisDatabase{
		client: *client,
		logger: logger,
	}
}

func (r *RedisDatabase) IsNotificationSent(eventID string) (bool, error) {
	exists, err := r.client.Exists(context.Background(), eventID).Result()
	return exists > 0, err
}

func (r *RedisDatabase) MarkNotificationSent(eventID string, ttl time.Duration) error {
	r.logger.Infof("Notification sent for event_ID: %s", eventID)
	return r.client.Set(context.Background(), eventID, "1", ttl).Err()
}
