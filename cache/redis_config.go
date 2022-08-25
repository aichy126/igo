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

func (rc *redisConfig) parse(conf map[string]interface{}) error {
	var ok bool

	//Address
	_, ok = conf["address"].(string)
	rc.Address = ""
	if ok {
		rc.Address = conf["address"].(string)
	}

	//Password
	_, ok = conf["password"].(string)
	rc.Password = ""
	if ok {
		rc.Password = conf["password"].(string)
	}

	//DB
	_, ok = conf["db"].(int64)
	rc.DB = 0
	if ok {
		rc.DB = int(conf["db"].(int64))
	}
	//PoolSize
	_, ok = conf["poolsize"].(int64)
	rc.PoolSize = 50
	if ok {
		rc.PoolSize = int(conf["poolsize"].(int64))
	}

	rc.normalize()
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
