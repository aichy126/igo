package cache

import (
	"strings"
	"testing"

	"github.com/aichy126/igo/config"
	"github.com/spf13/viper"
)

func newTestConfig(t *testing.T, toml string) *config.Config {
	t.Helper()
	v := viper.New()
	v.SetConfigType("toml")
	if err := v.ReadConfig(strings.NewReader(toml)); err != nil {
		t.Fatalf("读取测试配置失败: %v", err)
	}
	v.SetEnvPrefix("IGO")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	return &config.Config{Viper: v}
}

const redisTOML = `
[redis.default]
address  = "127.0.0.1:6379"
password = "file-password"
db       = 0
poolsize = 50
`

// redis 与 db 同因：UnmarshalKey 把整段配置解码到结构体，环境变量不会被合并进去。
// 密码恰恰是最不该写进配置文件、最需要用环境变量注入的东西。
func TestRedisApplyEnvOverride(t *testing.T) {
	t.Setenv("IGO_REDIS_DEFAULT_ADDRESS", "redis-host:6380")
	t.Setenv("IGO_REDIS_DEFAULT_PASSWORD", "env-password")
	conf := newTestConfig(t, redisTOML)

	rc := &redisConfig{Address: "127.0.0.1:6379", Password: "file-password", DB: 0, PoolSize: 50}
	applyEnvOverride(conf, "redis.default", rc)

	if rc.Address != "redis-host:6380" {
		t.Errorf("address 未被覆盖，实际: %s", rc.Address)
	}
	if rc.Password != "env-password" {
		t.Errorf("password 未被覆盖，实际: %s", rc.Password)
	}
	if rc.PoolSize != 50 {
		t.Errorf("未设环境变量的字段不应改变，实际 poolsize=%d", rc.PoolSize)
	}
}

func TestRedisApplyEnvOverrideNoEnvKeepsFileValues(t *testing.T) {
	conf := newTestConfig(t, redisTOML)

	rc := &redisConfig{Address: "127.0.0.1:6379", Password: "file-password", DB: 0, PoolSize: 50}
	before := *rc
	applyEnvOverride(conf, "redis.default", rc)

	if *rc != before {
		t.Errorf("无环境变量时配置不应改变\n修改前: %+v\n修改后: %+v", before, *rc)
	}
}

// 空环境变量【不会】覆盖配置文件的值：viper 的 AllowEmptyEnv 默认关闭，
// 空值被当作「未设置」。这是 viper 的既有行为，本次修复不改变它——
// 想要空密码请直接删掉配置文件里的那一行，而不是 export 一个空变量。
// 这条测试把这个行为钉住，免得日后有人误以为空变量能清值。
func TestRedisEmptyEnvDoesNotClearValue(t *testing.T) {
	t.Setenv("IGO_REDIS_DEFAULT_PASSWORD", "")
	conf := newTestConfig(t, redisTOML)

	rc := &redisConfig{Password: "file-password"}
	applyEnvOverride(conf, "redis.default", rc)

	if rc.Password != "file-password" {
		t.Errorf("空环境变量不应改变配置文件的值，实际: %q", rc.Password)
	}
}

// db=0 是合法值，不能用「>0 才覆盖」的判断，否则切回 0 号库就做不到。
func TestRedisDBZeroIsRespected(t *testing.T) {
	t.Setenv("IGO_REDIS_DEFAULT_DB", "0")
	conf := newTestConfig(t, "\n[redis.default]\naddress = \"127.0.0.1:6379\"\ndb = 3\n")

	rc := &redisConfig{DB: 3}
	applyEnvOverride(conf, "redis.default", rc)

	if rc.DB != 0 {
		t.Errorf("db 应能被环境变量设为 0，实际 %d", rc.DB)
	}
}
