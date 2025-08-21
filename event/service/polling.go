package service

import "olympsis-server/redis"

type PollingService struct {
	cache redis.RedisDatabase
}
