package cache

import (
	"encoding/json"
	"strings"

	"github.com/aichy126/igo/config"
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

// applyEnvOverride 让 IGO_ 环境变量对 redis 配置生效。
//
// 与 db 包同因：viper 的 AutomaticEnv 只在按【完整 key】调用 Get 系列方法时生效，
// 而 UnmarshalKey 是把整段配置解码到结构体，环境变量不会被合并进去
// （viper 的已知行为）。缺了这一步，IGO_REDIS_DEFAULT_PASSWORD 这类覆盖会被静默忽略——
// 而密码恰恰是最不该写进配置文件、最需要用环境变量注入的东西。
//
// 只在环境变量给出有效值时才覆盖：GetString 在环境变量缺失时会回落到配置文件的值，
// 所以判空既挡住了空环境变量，也保证无环境变量时行为不变。
//
// 注意 viper 的 AllowEmptyEnv 默认关闭，空环境变量会被当作「未设置」——
// 因此无法用空变量把配置文件里的密码清掉，需要空密码请直接删掉配置里那一行。
// password 与 db 走 IsSet 而非判空，是为了让「显式配成空串 / 0 号库」也能生效。
func applyEnvOverride(conf *config.Config, prefix string, rc *redisConfig) {
	if v := strings.TrimSpace(conf.GetString(prefix + ".address")); v != "" {
		rc.Address = v
	}
	if conf.IsSet(prefix + ".password") {
		rc.Password = conf.GetString(prefix + ".password")
	}
	if conf.IsSet(prefix + ".db") {
		rc.DB = conf.GetInt(prefix + ".db")
	}
	if v := conf.GetInt(prefix + ".poolsize"); v > 0 {
		rc.PoolSize = v
	}
	if v := conf.GetInt(prefix + ".dial_timeout"); v > 0 {
		rc.DialTimeout = v
	}
	if v := conf.GetInt(prefix + ".read_timeout"); v > 0 {
		rc.ReadTimeout = v
	}
	if v := conf.GetInt(prefix + ".write_timeout"); v > 0 {
		rc.WriteTimeout = v
	}
}
