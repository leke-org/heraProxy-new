package common

import (
	"sync"

	"github.com/redis/go-redis/v9"
	"proxy_server/config"
)

var GetRedisDB = sync.OnceValue[*redis.Client](func() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:         config.GetConf().Redis.Addr,
		Password:     config.GetConf().Redis.Password,
		DB:           config.GetConf().Redis.DB,
		MinIdleConns: config.GetConf().Redis.MinIdleConns,
		MaxIdleConns: config.GetConf().Redis.MaxIdleConns,
		PoolSize:     config.GetConf().Redis.PoolSize,
	})
	return rdb
})
