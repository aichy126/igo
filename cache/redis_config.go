package cache

import (
	"encoding/json"
	"strings"

	"github.com/aichy126/igo/log"

	"github.com/redis/go-redis/v9"

	"time"
)

type redisConfig struct {
	Address      string `json:"address" toml:"address" mapstructure:"address"`
	Password     string `json:"password" toml:"password" mapstructure:"password"`
	DB           int    `json:"db" toml:"db" mapstructure:"db"`
	PoolSize     int    `json:"poolsize" toml:"poolsize" mapstructure:"poolsize"`
	DialTimeout  int    `json:"dial_timeout" toml:"dial_timeout" mapstructure:"dial_timeout"`    // 毫秒
	ReadTimeout  int    `json:"read_timeout" toml:"read_timeout" mapstructure:"read_timeout"`    // 毫秒
	WriteTimeout int    `json:"write_timeout" toml:"write_timeout" mapstructure:"write_timeout"` // 毫秒
}

func (rc redisConfig) String() string {
	data, _ := json.Marshal(rc)
	return string(data)
}

func (rc *redisConfig) parse(conf *redisConfig) error {
	rc.Address = strings.TrimSpace(conf.Address)
	rc.Password = strings.TrimSpace(conf.Password)
	rc.DB = conf.DB
	rc.PoolSize = conf.PoolSize
	rc.DialTimeout = conf.DialTimeout
	rc.ReadTimeout = conf.ReadTimeout
	rc.WriteTimeout = conf.WriteTimeout
	if !strings.Contains(rc.Address, ":") {
		log.Warn("redis address 未指定端口,使用默认端口 6379", log.Any("address", rc.Address))
		rc.Address = rc.Address + ":6379"
	}
	return nil
}

func (rc redisConfig) toOptions() *redis.Options {
	options := &redis.Options{
		Addr:         rc.Address,
		Password:     rc.Password,
		DB:           rc.DB,
		PoolSize:     rc.PoolSize,
		DialTimeout:  time.Duration(rc.DialTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(rc.WriteTimeout) * time.Millisecond,
		ReadTimeout:  time.Duration(rc.ReadTimeout) * time.Millisecond,
	}
	return options
}

func (rc *redisConfig) newRedis() (*Redis, error) {
	options := rc.toOptions()
	client := redis.NewClient(options)
	return NewRedis(client, options), nil
}
