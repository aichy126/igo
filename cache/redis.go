package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aichy126/igo/config"
)

type RedisManager struct {
	mutex     sync.RWMutex
	resources map[string]*Redis
}

// NewRedisManager 从配置初始化所有 redis 连接
// 配置了的 redis 必须连接成功(ping),否则返回错误(fail-fast);
// 完全没有 [redis] 配置时返回空 manager,不报错。
func NewRedisManager(conf *config.Config) (*RedisManager, error) {
	r := &RedisManager{
		resources: make(map[string]*Redis),
	}
	if err := r.initRedis(conf); err != nil {
		return nil, err
	}
	return r, nil
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

	return r, fmt.Errorf("redis 配置 [%s] 不存在,请检查配置文件中的 [redis.%s] 配置", redisConfigCenterKey, redisConfigCenterKey)
}

// Names 返回所有已初始化的 redis 配置名
func (rm *RedisManager) Names() []string {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	names := make([]string, 0, len(rm.resources))
	for name := range rm.resources {
		names = append(names, name)
	}
	return names
}

// PingAll 对所有 redis 执行 Ping,返回每个实例的结果(nil 表示正常)
func (rm *RedisManager) PingAll(ctx context.Context) map[string]error {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	result := make(map[string]error, len(rm.resources))
	for name, r := range rm.resources {
		result[name] = r.Ping(ctx).Err()
	}
	return result
}

func (rm *RedisManager) initRedis(conf *config.Config) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	redisList := make(map[string]*redisConfig)
	if err := conf.UnmarshalKey("redis", &redisList); err != nil {
		return fmt.Errorf("redis 配置解析失败: %w", err)
	}

	for name, itemRedisConfig := range redisList {
		var rc redisConfig
		if err := rc.parse(itemRedisConfig); err != nil {
			return fmt.Errorf("redis 配置 [redis.%s] 解析失败: %w", name, err)
		}
		r, err := rm.newRedis(rc)
		if err != nil {
			return fmt.Errorf("redis [%s] 初始化失败: %w", name, err)
		}
		// 启动时验证连通性,连不上尽早暴露
		pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err = r.Ping(pingCtx).Err()
		cancel()
		if err != nil {
			return fmt.Errorf("redis [%s] 连接失败(ping %s): %w", name, rc.Address, err)
		}
		rm.resources[name] = r
	}
	return nil
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
				fmt.Printf("关闭Redis连接失败 name=%s error=%v\n", name, err)
			}
		}
	}
	return nil
}
