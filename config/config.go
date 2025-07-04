package config

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/spf13/viper"
)

const EnvConfigPath = "CONFIG_PATH"
const EnvConfigAddress = "CONFIG_ADDRESS"
const EnvConfigKEY = "CONFIG_KEY"

// Web
type Config struct {
	*viper.Viper
	mu sync.RWMutex
	// 配置变更回调
	changeCallbacks []func()
}

func NewConfig(ConfigPath string) (*Config, error) {
	Conf := new(Config)
	ConfigFilePath := ConfigPath
	if len(ConfigFilePath) < 1 {
		ConfigFilePath = GetLocalConfigPath()
	}
	// another chance to read the config file from environment variable
	if ConfigFilePath == "" {
		ConfigFilePath = os.Getenv(EnvConfigPath)
	}

	localConfig := viper.New()
	localConfig.SetConfigFile(ConfigFilePath)
	err := localConfig.ReadInConfig() // Find and read the config file
	if err == nil {                   // Handle errors reading the config file
		Conf.Viper = localConfig
	}
	var address, key string
	exist := localConfig.IsSet("config")
	if exist {
		address = localConfig.GetString("config.address")
		key = localConfig.GetString("config.key")
	}
	if address == "" || key == "" {
		address = os.Getenv(EnvConfigAddress)
		key = os.Getenv(EnvConfigKEY)
	}
	if address == "" || key == "" {
		return Conf, err
	}
	consulConfig, err := getConsulConf(address, key)
	if err != nil {
		return Conf, err
	} else {
		Conf.Viper = consulConfig
	}
	return Conf, err
}

func getConsulConf(address, key string) (*viper.Viper, error) {
	consulConfig := viper.New()
	consulConfig.SetConfigType("toml")
	if len(address) > 0 && len(key) > 0 {
		newConfigClient(address)
		t, err := GetByTree(key)
		if err != nil {
			return consulConfig, err
		}
		err = consulConfig.ReadConfig(bytes.NewBuffer(t))
		if err != nil {
			return consulConfig, err
		}
	}
	return consulConfig, nil
}

var client *consulapi.Client

func newConfigClient(address string) {
	config := consulapi.DefaultConfig()
	config.Address = address
	c, err := consulapi.NewClient(config)
	if err != nil {
		panic(fmt.Sprintf("consul client error : %+v\n ", err))
	}
	client = c
}

func GetByTree(key string) ([]byte, error) {
	kvPair, _, err := client.KV().Get(key, &consulapi.QueryOptions{})
	if err != nil {
		return nil, err
	}
	if kvPair == nil {
		return nil, fmt.Errorf("consul key not found:%s", key)
	}
	return kvPair.Value, nil
}

// Get the local config file path from the command line
func GetLocalConfigPath() string {
	confPath := flag.String("c", "config.toml", "configure file")
	flag.Parse()
	return *confPath
}

// Validate 验证配置
func (c *Config) Validate() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 验证必要的配置项
	requiredFields := []string{
		"local.address",
		"local.debug",
	}

	for _, field := range requiredFields {
		if !c.IsSet(field) {
			return fmt.Errorf("缺少必要的配置项: %s", field)
		}
	}

	return nil
}

// WatchConfig 监听配置变更
func (c *Config) WatchConfig() {
	c.Viper.WatchConfig()
	c.Viper.OnConfigChange(func(e fsnotify.Event) {
		c.mu.Lock()
		defer c.mu.Unlock()

		// 执行配置变更回调
		for _, callback := range c.changeCallbacks {
			callback()
		}
	})
}

// AddChangeCallback 添加配置变更回调
func (c *Config) AddChangeCallback(callback func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.changeCallbacks = append(c.changeCallbacks, callback)
}

// GetWithDefault 获取配置值，如果不存在则返回默认值
func (c *Config) GetWithDefault(key string, defaultValue interface{}) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.IsSet(key) {
		return c.Get(key)
	}
	return defaultValue
}

// GetStringWithDefault 获取字符串配置值，如果不存在则返回默认值
func (c *Config) GetStringWithDefault(key, defaultValue string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.IsSet(key) {
		return c.GetString(key)
	}
	return defaultValue
}

// GetIntWithDefault 获取整数配置值，如果不存在则返回默认值
func (c *Config) GetIntWithDefault(key string, defaultValue int) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.IsSet(key) {
		return c.GetInt(key)
	}
	return defaultValue
}

// GetBoolWithDefault 获取布尔配置值，如果不存在则返回默认值
func (c *Config) GetBoolWithDefault(key string, defaultValue bool) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.IsSet(key) {
		return c.GetBool(key)
	}
	return defaultValue
}
