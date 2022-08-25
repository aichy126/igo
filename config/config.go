package config

import (
	"flag"
	"os"

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
	return Conf, nil
}

func InitConfig(ConfigFilePath string) (Config *viper.Viper, err error) {
	// another chance to read the config file from environment variable
	if ConfigFilePath == "" {
		ConfigFilePath = os.Getenv(EnvConfigPath)
	}

	localConfig := viper.New()
	localConfig.SetConfigFile(ConfigFilePath)
	err = localConfig.ReadInConfig() // Find and read the config file
	if err != nil {                  // Handle errors reading the config file
		return Config, err
	}
	Config = localConfig
	return Config, nil
}

// Get the local config file path from the command line
func GetLocalConfigPath() string {
	confPath := flag.String("c", "config.toml", "configure file")
	flag.Parse()
	return *confPath
}
