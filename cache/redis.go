package cache

import (
	"fmt"
	"sync"

	"github.com/aichy126/igo/config"
	"github.com/aichy126/igo/log"
)

type RedisManager struct {
	mutex     sync.RWMutex
	resources map[string]*Redis
}

func NewRedisManager(conf *config.Config) *RedisManager {
	r := &RedisManager{
		resources: make(map[string]*Redis),
	}
	r.initRedis(conf)
	return r
}

func (rm *RedisManager) Get(redisConfigCenterKey string) (*Redis, error) {
	return rm.get(redisConfigCenterKey)
}

func (rm *RedisManager) get(redisConfigCenterKey string) (r *Redis, err error) {

	rm.mutex.RLock()
	r, ok := rm.resources[redisConfigCenterKey]
	rm.mutex.RUnlock()
	if ok {
		return r, nil
	}

	return r, fmt.Errorf("redis does not exist")
}

func (rm *RedisManager) initRedis(conf *config.Config) (err error) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	var rc redisConfig
	redisList := make(map[string]*redisConfig, 0)
	err = conf.UnmarshalKey("redis", &redisList)
	if err != nil {
		return
	}

	for name, itemRedisConfig := range redisList {
		err = rc.parse(itemRedisConfig)
		if err != nil {
			log.Error("initRedis", log.Any("error", err.Error()))
			continue
		}
		r, err := rm.newRedis(rc)
		if err != nil {
			log.Error("initRedis", log.Any("error", err.Error()))
			continue
		}
		rm.resources[name] = r
	}
	return

}

func (rm *RedisManager) newRedis(config redisConfig) (*Redis, error) {
	for _, r := range rm.resources {
		if r.IsEqual(config.toOptions()) {
			return r, nil
		}
	}
	return config.newRedis()
}

// Close 关闭所有Redis连接
func (rm *RedisManager) Close() error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	for name, r := range rm.resources {
		if r != nil && r.Client != nil {
			if err := r.Client.Close(); err != nil {
				log.Error("关闭Redis连接失败", log.Any("name", name), log.Any("error", err))
			}
		}
	}
	return nil
}
