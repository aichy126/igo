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

	redisList := conf.GetStringMap("redis")

	for name, itemRedisConfig := range redisList {
		_, ok := itemRedisConfig.(map[string]interface{})
		if !ok {
			continue
		}
		err = rc.parse(itemRedisConfig.(map[string]interface{}))
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
