package db

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// newTestViper 构造一个与 config 包同款设置的 viper：TOML + IGO_ 前缀环境变量。
func newTestViper(t *testing.T, toml string) *viper.Viper {
	t.Helper()
	v := viper.New()
	v.SetConfigType("toml")
	if err := v.ReadConfig(strings.NewReader(toml)); err != nil {
		t.Fatalf("读取测试配置失败: %v", err)
	}
	v.SetEnvPrefix("IGO")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	return v
}

const dbTOML = `
[mysql.default]
max_idle = 5
max_open = 10
is_debug = false
data_source = "file-user:file-pass@tcp(127.0.0.1:3306)/from_file"
`

// 这是本次修复要解决的核心问题：GetStringMap 读出来的是配置文件里的原始 map，
// 环境变量不会被合并进去，导致 IGO_MYSQL_DEFAULT_DATA_SOURCE 被静默忽略。
func TestApplyEnvOverrideDataSource(t *testing.T) {
	t.Setenv("IGO_MYSQL_DEFAULT_DATA_SOURCE", "env-user:env-pass@tcp(db:3306)/from_env")
	v := newTestViper(t, dbTOML)

	c := &DBConfig{Datasource: "file-user:file-pass@tcp(127.0.0.1:3306)/from_file", MaxIdle: 5, MaxOpen: 10}
	applyEnvOverride(v, "mysql.default", c)

	if want := "env-user:env-pass@tcp(db:3306)/from_env"; c.Datasource != want {
		t.Errorf("data_source 未被环境变量覆盖\n实际: %s\n期望: %s", c.Datasource, want)
	}
	// 未设环境变量的字段应保持配置文件的值
	if c.MaxIdle != 5 || c.MaxOpen != 10 {
		t.Errorf("未被覆盖的字段不应改变，实际 max_idle=%d max_open=%d", c.MaxIdle, c.MaxOpen)
	}
}

// 没有环境变量时行为必须与修复前完全一致，否则这个改动会影响所有现有部署。
func TestApplyEnvOverrideNoEnvKeepsFileValues(t *testing.T) {
	v := newTestViper(t, dbTOML)

	c := &DBConfig{
		Datasource: "file-user:file-pass@tcp(127.0.0.1:3306)/from_file",
		MaxIdle:    5, MaxOpen: 10, MaxIdleLife: 0, IsDebug: false,
	}
	before := *c
	applyEnvOverride(v, "mysql.default", c)

	if *c != before {
		t.Errorf("无环境变量时配置不应改变\n修改前: %+v\n修改后: %+v", before, *c)
	}
}

// 空的环境变量不该把配置文件里的有效值清掉——否则一个手滑的 export 会让服务连不上库。
func TestApplyEnvOverrideEmptyEnvIsIgnored(t *testing.T) {
	t.Setenv("IGO_MYSQL_DEFAULT_DATA_SOURCE", "")
	v := newTestViper(t, dbTOML)

	c := &DBConfig{Datasource: "file-user:file-pass@tcp(127.0.0.1:3306)/from_file"}
	applyEnvOverride(v, "mysql.default", c)

	if !strings.Contains(c.Datasource, "from_file") {
		t.Errorf("空环境变量不应覆盖配置文件的值，实际: %s", c.Datasource)
	}
}

func TestApplyEnvOverrideNumericAndBool(t *testing.T) {
	t.Setenv("IGO_MYSQL_DEFAULT_MAX_IDLE", "42")
	t.Setenv("IGO_MYSQL_DEFAULT_IS_DEBUG", "true")
	v := newTestViper(t, dbTOML)

	c := &DBConfig{MaxIdle: 5, IsDebug: false}
	applyEnvOverride(v, "mysql.default", c)

	if c.MaxIdle != 42 {
		t.Errorf("max_idle 未被覆盖，实际 %d", c.MaxIdle)
	}
	if !c.IsDebug {
		t.Error("is_debug 未被覆盖为 true")
	}
}

// sqlite 与 mysql 走同一段逻辑，前缀不同不应互相干扰。
func TestApplyEnvOverrideSqlitePrefix(t *testing.T) {
	t.Setenv("IGO_SQLITE_DEFAULT_DATA_SOURCE", "/data/env.db")
	v := newTestViper(t, "\n[sqlite.default]\ndata_source = \"file.db\"\n")

	c := &DBConfig{Datasource: "file.db"}
	applyEnvOverride(v, "sqlite.default", c)

	if c.Datasource != "/data/env.db" {
		t.Errorf("sqlite data_source 未被覆盖，实际: %s", c.Datasource)
	}
}
