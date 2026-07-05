package config

import (
	"fmt"
	"os"
	"testing"
)

func TestInitConfig(t *testing.T) {
	configPath := os.Getenv(EnvConfigPath)
	if configPath == "" {
		configPath = "./config.toml"
	}
	config, err := NewConfig(configPath)
	if err != nil {
		panic(err)
	}
	fmt.Println(config.GetString("local.address"))
}

// TestEnvOverride 验证 IGO_ 前缀环境变量覆盖配置文件
func TestEnvOverride(t *testing.T) {
	t.Setenv("IGO_LOCAL_ADDRESS", ":9999")
	dir := t.TempDir()
	path := dir + "/config.toml"
	if err := os.WriteFile(path, []byte("[local]\naddress = \":8001\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	conf, err := NewConfig(path)
	if err != nil {
		t.Fatalf("NewConfig error: %v", err)
	}
	if got := conf.GetString("local.address"); got != ":9999" {
		t.Errorf("环境变量应覆盖配置文件: got %q, want :9999", got)
	}
}
