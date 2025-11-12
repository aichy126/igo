package config

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/spf13/viper"
)

const EnvConfigPath = "CONFIG_PATH"
const EnvConfigAddress = "CONFIG_ADDRESS"
const EnvConfigKEY = "CONFIG_KEY"

// Config 配置管理
type Config struct {
	*viper.Viper
	mu              sync.RWMutex
	// 配置变更回调
	changeCallbacks []func()
	// 热重载开关
	hotReloadEnabled bool
	// 配置文件路径
	configPath string
	// 配置源类型
	configSource string // "file", "consul", "env"
	// Consul配置
	consulAddress string
	consulKey     string
	// 热重载轮询间隔（秒）- 仅对Consul/env生效，-1或0表示禁用
	hotReloadInterval int
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

	// 保存配置文件路径用于热重载
	Conf.configPath = ConfigFilePath
	Conf.configSource = "file" // 默认为文件配置

	localConfig := viper.New()
	localConfig.SetConfigFile(ConfigFilePath)
	err := localConfig.ReadInConfig() // Find and read the config file
	if err == nil {                   // Handle errors reading the config file
		Conf.Viper = localConfig
		// 只有文件配置才默认启用热重载
		Conf.hotReloadEnabled = true
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
	if address != "" && key != "" {
		// 使用Consul配置
		Conf.configSource = "consul"
		Conf.consulAddress = address
		Conf.consulKey = key
		Conf.hotReloadEnabled = false    // Consul配置默认不启用热重载
		Conf.hotReloadInterval = 0       // 默认不轮询

		consulConfig, err := getConsulConf(address, key)
		if err != nil {
			return Conf, err
		} else {
			Conf.Viper = consulConfig
		}
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

// triggerCallbacks 触发配置变更回调
func (c *Config) triggerCallbacks() {
	for _, callback := range c.changeCallbacks {
		go func(cb func()) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("配置变更回调执行失败: %v\n", r)
				}
			}()
			cb()
		}(callback)
	}
}

// watchConsulConfig 监听Consul配置变更（用户自定义轮询间隔）
func (c *Config) watchConsulConfig() {
	interval := time.Duration(c.hotReloadInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fmt.Printf("Consul配置热重载已启用，轮询间隔: %d秒\n", c.hotReloadInterval)

	for range ticker.C {
		if !c.IsHotReloadEnabled() || c.hotReloadInterval <= 0 {
			fmt.Printf("Consul配置热重载已停止\n")
			return // 如果热重载被禁用或间隔无效，停止监听
		}

		// 从Consul获取最新配置
		newConfig, err := getConsulConf(c.consulAddress, c.consulKey)
		if err != nil {
			fmt.Printf("Consul配置获取失败: %v\n", err)
			continue
		}

		c.mu.Lock()
		// 比较配置是否有变更（简单的字符串比较）
		oldStr := c.Viper.AllSettings()
		newStr := newConfig.AllSettings()

		configChanged := false
		if len(oldStr) != len(newStr) {
			configChanged = true
		} else {
			// 简单检查主要配置项是否变更
			for key := range oldStr {
				if fmt.Sprintf("%v", oldStr[key]) != fmt.Sprintf("%v", newStr[key]) {
					configChanged = true
					break
				}
			}
		}

		if configChanged {
			c.Viper = newConfig
			fmt.Printf("Consul配置已更新\n")
			c.mu.Unlock()
			c.triggerCallbacks()
		} else {
			c.mu.Unlock()
		}
	}
}

// SetHotReloadInterval 设置热重载轮询间隔（仅对Consul配置有效）
// intervalSeconds: 轮询间隔秒数，-1或0表示禁用热重载，>0表示启用并设置间隔
func (c *Config) SetHotReloadInterval(intervalSeconds int) *Config {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.configSource == "file" {
		// 文件配置忽略此设置，总是启用热重载
		fmt.Printf("文件配置自动启用热重载，忽略轮询间隔设置\n")
		return c
	}

	if c.configSource == "consul" {
		if intervalSeconds > 0 {
			c.hotReloadEnabled = true
			c.hotReloadInterval = intervalSeconds
			fmt.Printf("Consul配置热重载已设置，轮询间隔: %d秒\n", intervalSeconds)
		} else {
			c.hotReloadEnabled = false
			c.hotReloadInterval = 0
			fmt.Printf("Consul配置热重载已禁用\n")
		}
	} else {
		fmt.Printf("当前配置源(%s)不支持热重载\n", c.configSource)
	}

	return c
}

// EnableHotReload 启用热重载（文件配置专用）
func (c *Config) EnableHotReload() *Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hotReloadEnabled = true
	return c
}

// DisableHotReload 禁用热重载
func (c *Config) DisableHotReload() *Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hotReloadEnabled = false
	return c
}

// IsHotReloadEnabled 检查是否启用了热重载
func (c *Config) IsHotReloadEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hotReloadEnabled
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 只验证最基本的配置项
	if !c.IsSet("local.address") {
		return fmt.Errorf("缺少必要的配置项: local.address")
	}
	return nil
}

// WatchConfig 监听配置变更（只有启用热重载时才生效）
func (c *Config) WatchConfig() {
	if !c.IsHotReloadEnabled() {
		return // 如果没有启用热重载，直接返回
	}

	switch c.configSource {
	case "file":
		// 文件配置使用fsnotify监听（IO影响很小），忽略轮询间隔设置
		c.Viper.WatchConfig()
		c.Viper.OnConfigChange(func(e fsnotify.Event) {
			c.mu.Lock()
			defer c.mu.Unlock()

			// 重新读取配置
			if err := c.Viper.ReadInConfig(); err != nil {
				fmt.Printf("文件配置热重载失败: %v\n", err)
				return
			}

			fmt.Printf("文件配置已重新加载: %s\n", e.Name)
			c.triggerCallbacks()
		})

	case "consul":
		// Consul配置使用用户指定的轮询间隔
		if c.hotReloadInterval > 0 {
			go c.watchConsulConfig()
		} else {
			fmt.Printf("Consul配置热重载未启用（轮询间隔: %d秒）\n", c.hotReloadInterval)
		}

	default:
		fmt.Printf("不支持的配置源: %s\n", c.configSource)
	}
}

// AddChangeCallback 添加配置变更回调
func (c *Config) AddChangeCallback(callback func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.changeCallbacks = append(c.changeCallbacks, callback)
}

// RemoveAllCallbacks 清除所有配置变更回调
func (c *Config) RemoveAllCallbacks() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.changeCallbacks = nil
}

// ReloadConfig 手动重新加载配置
func (c *Config) ReloadConfig() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.Viper.ReadInConfig(); err != nil {
		return fmt.Errorf("重新加载配置失败: %w", err)
	}

	fmt.Printf("配置已手动重新加载: %s\n", c.configPath)

	// 执行配置变更回调
	for _, callback := range c.changeCallbacks {
		go func(cb func()) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("配置变更回调执行失败: %v\n", r)
				}
			}()
			cb()
		}(callback)
	}

	return nil
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
