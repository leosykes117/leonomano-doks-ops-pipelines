package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
)

type ConfigLog struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Color  bool   `mapstructure:"color"`
}

type TerragruntConfig struct {
	RootPath        string `mapstructure:"root_path"`
	BaseModulesPath string `mapstructure:"base_modules_path"`
	Log             ConfigLog
}

type Config struct {
	Log            ConfigLog        `mapstructure:"log"`
	Terragrunt     TerragruntConfig `mapstructure:"terragrunt"`
	DryRun         bool             `mapstructure:"dry-run"`
	Environment    string
	moduleRootPath string
}

// Load reads config from ./configs/config.yaml,
// env vars and (optionally) flags.
func Load() (*Config, error) {
	v := viper.New()
	c := &Config{}
	moduleRootPath, err := getModuleRootPath()
	if err != nil {
		return nil, fmt.Errorf("get module root path: %w", err)
	}

	// default values
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "console")
	v.SetDefault("log.color", true)
	v.SetDefault("terragrunt.log.color", true)
	v.SetDefault("dry-run", false)

	// allow ENV variables like:
	//   LOG_LEVEL=debug
	//   TERRAGRUNT_ROOT_PATH=/infra/envs/dev
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// look for file: configs/config.yaml
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.SetConfigType("yml")
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")

	// optional: if no config file present, continue
	if err := v.ReadInConfig(); err != nil {
		fmt.Println("[config] no config file found, using defaults + env")
	}

	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	c.moduleRootPath = moduleRootPath

	if c.Log.Level == "debug" {
		log.Printf("Config Loaded")
		log.Printf("config=%+v", c)
	}

	return c, nil
}
