package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	APIKey      string  `json:"api_key"`
	BaseURL     string  `json:"base_url"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	NumPredict  int     `json:"num_predict"`
	Theme       string  `json:"theme"`
}

var DefaultConfig = Config{
	BaseURL:     "https://ollama.com",
	Model:       "qwen3.5:cloud",
	Temperature: 0.7,
	NumPredict:  4096,
	Theme:       "dark",
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}
	dir := filepath.Join(home, ".agenttea")
	return dir, nil
}

func ConfigPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig
			if apiKey := os.Getenv("OLLAMA_API_KEY"); apiKey != "" {
				cfg.APIKey = apiKey
			}
			return &cfg, nil
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("OLLAMA_API_KEY")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultConfig.BaseURL
	}
	if cfg.Model == "" {
		cfg.Model = DefaultConfig.Model
	}
	if cfg.Temperature == 0 {
		cfg.Temperature = DefaultConfig.Temperature
	}
	if cfg.NumPredict == 0 {
		cfg.NumPredict = DefaultConfig.NumPredict
	}
	if cfg.Theme == "" {
		cfg.Theme = DefaultConfig.Theme
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	path := filepath.Join(dir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}
