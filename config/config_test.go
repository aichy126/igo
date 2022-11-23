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
