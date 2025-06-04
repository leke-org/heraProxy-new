package util

import "github.com/coocood/freecache"

const (
	// 缓存大小 100MB
	cacheSize = 100 * 1024 * 1024
)

var cache *freecache.Cache

func init() {
	cache = freecache.NewCache(cacheSize)
}

func GetOrSetFreeCache(key string, value []byte, expireSeconds int) (retValue []byte, err error) {
	return cache.GetOrSet([]byte(key), value, expireSeconds)
}

func GetFreeCache(key string) ([]byte, error) {
	value, err := cache.Get([]byte(key))
	if err != nil {
		return nil, err
	}
	return value, nil
}

func SetFreeCache(key string, value []byte, expireSeconds int) error {
	return cache.Set([]byte(key), value, expireSeconds)
}
