package cache

import (
	"fmt"

	"github.com/aichy126/igo/config"
	"github.com/redis/go-redis/v9"
)

// Cache
type Cache struct {
	*RedisManager
}

// NewCache 从配置初始化缓存
// 配置了的 redis 连接失败会返回错误(fail-fast);完全没配置 redis 时返回可用的空实例
func NewCache(conf *config.Config) (*Cache, error) {
	cache := new(Cache)
	manager, err := NewRedisManager(conf)
	if err != nil {
		return nil, err
	}
	cache.RedisManager = manager
	return cache, nil
}

type Redis struct {
	*redis.Client
	State   RedisState `json:"state"`
	Options *redis.Options
}
type RedisState int

const (
	ActiveServer = RedisState(0)
	DownServer   = RedisState(1)
)

func NewRedis(client *redis.Client, options *redis.Options) *Redis {
	return &Redis{
		Client:  client,
		Options: options,
		State:   ActiveServer,
	}
}

func (r Redis) IsEqual(options *redis.Options) bool {
	if options == nil {
		return false
	}
	if r.Options == nil {
		return false
	}

	if r.Options.Addr == options.Addr && r.Options.DB == options.DB {
		return true
	}
	return false
}

func (this *Redis) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`{"state":%d,"address":"%s","db":%d}`, this.State, this.Options.Addr, this.Options.DB)), nil
}

func (r *Redis) GetClient() *redis.Client {
	return r.Client
}

// Close 关闭所有缓存连接
func (c *Cache) Close() error {
	if c.RedisManager != nil {
		return c.RedisManager.Close()
	}
	return nil
}
