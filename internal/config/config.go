package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	LLM      LLMConfig      `mapstructure:"llm"`
	Database DatabaseConfig `mapstructure:"database"`
	Queue    QueueConfig    `mapstructure:"queue"`
	Server   ServerConfig   `mapstructure:"server"`
	Log      LogConfig      `mapstructure:"log"`
}

type LLMConfig struct {
	Provider         string `mapstructure:"provider"`
	Model            string `mapstructure:"model"`
	Host             string `mapstructure:"host"`
	APIKey           string `mapstructure:"api_key"`
	TimeoutSeconds   int    `mapstructure:"timeout_seconds"`
	FallbackProvider string `mapstructure:"fallback_provider"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type QueueConfig struct {
	PollIntervalSeconds int     `mapstructure:"poll_interval_seconds"`
	MaxRetries          int     `mapstructure:"max_retries"`
	BackoffMultiplier   float64 `mapstructure:"backoff_multiplier"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".todoloo")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)
	viper.AddConfigPath(".")

	err = viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	err = viper.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
