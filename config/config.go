package config

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/spf13/viper"
)

const EnvConfigPath = "CONFIG_PATH"

// Web
type Config struct {
	*viper.Viper
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
	if err != nil {                   // Handle errors reading the config file
		return Conf, err
	}
	Conf.Viper = localConfig
	consulConfig, exist, err := getConsulConf(localConfig)
	if err != nil {
		return Conf, err
	}
	if exist {
		Conf.Viper = consulConfig
	}
	return Conf, nil
}

func getConsulConf(config *viper.Viper) (*viper.Viper, bool, error) {
	consulConfig := viper.New()
	consulConfig.SetConfigType("toml")
	exist := config.IsSet("config")
	if !exist {
		return consulConfig, exist, nil
	}
	address := config.GetString("config.address")
	key := config.GetString("config.key")
	if len(address) > 0 && len(key) > 0 {
		newConfigClient(address)
		t, err := GetByTree(key)
		if err != nil {
			return consulConfig, exist, err
		}
		err = consulConfig.ReadConfig(bytes.NewBuffer(t))
		if err != nil {
			return consulConfig, exist, err
		}
	}
	return consulConfig, exist, nil
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
