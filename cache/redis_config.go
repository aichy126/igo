package cache

import (
	"encoding/json"
	"strings"

	"github.com/aichy126/igo/log"

	"github.com/go-redis/redis/v8"

	"time"
)

type redisConfig struct {
	Address      string `json:"address" toml:"address"`
	Password     string `json:"password" toml:"password"`
	DB           int    `json:"db" toml:"db"`
	PoolSize     int    `json:"poolsize" toml:"poolsize"`
	DialTimeout  int    `json:"DialTimeout" toml:"dial_timeout"`
	ReadTimeout  int    `json:"ReadTimeout" toml:"read_timeout"`
	WriteTimeout int    `json:"WriteTimeout" toml:write_timeout`
}

func (rc redisConfig) String() string {
	data, _ := json.Marshal(rc)
	return string(data)
}

func (rc *redisConfig) parse(conf *redisConfig) error {
	conf.normalize()
	return nil
}

func (rc *redisConfig) normalize() {
	rc.Address = strings.TrimSpace(rc.Address)
	rc.Password = strings.TrimSpace(rc.Password)
	if !strings.Contains(rc.Address, ":") {
		log.Error("redisconfig.address doesn't contains port", log.Any("rc.Address", rc.Address))

		rc.Address = rc.Address + ":6379"
	}
}

func (rc redisConfig) toOptions() *redis.Options {
	options := &redis.Options{
		Addr:         rc.Address,
		Password:     rc.Password,
		DB:           int(rc.DB),
		PoolSize:     int(rc.PoolSize),
		DialTimeout:  time.Duration(rc.DialTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(rc.WriteTimeout) * time.Millisecond,
		ReadTimeout:  time.Duration(rc.ReadTimeout) * time.Millisecond,
	}
	return options

}

func (rc *redisConfig) newRedis() (*Redis, error) {
	options := &redis.Options{
		Addr:         rc.Address,
		Password:     rc.Password,
		DB:           int(rc.DB),
		PoolSize:     int(rc.PoolSize),
		DialTimeout:  time.Duration(rc.DialTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(rc.WriteTimeout) * time.Millisecond,
		ReadTimeout:  time.Duration(rc.ReadTimeout) * time.Millisecond,
	}
	client := redis.NewClient(options)

	return NewRedis(client, options), nil
}
