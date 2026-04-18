package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig
	if cfg.BaseURL != "https://ollama.com" {
		t.Errorf("DefaultConfig.BaseURL = %q, want %q", cfg.BaseURL, "https://ollama.com")
	}
	if cfg.Model != "qwen3.5:cloud" {
		t.Errorf("DefaultConfig.Model = %q, want %q", cfg.Model, "qwen3.5:cloud")
	}
	if cfg.Temperature != 0.7 {
		t.Errorf("DefaultConfig.Temperature = %f, want %f", cfg.Temperature, 0.7)
	}
	if cfg.NumPredict != 4096 {
		t.Errorf("DefaultConfig.NumPredict = %d, want %d", cfg.NumPredict, 4096)
	}
	if cfg.Theme != "dark" {
		t.Errorf("DefaultConfig.Theme = %q, want %q", cfg.Theme, "dark")
	}
}

func TestLoadFromJSON(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, ".agenttea")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	fileConfig := Config{
		APIKey:      "file-api-key",
		BaseURL:     "https://custom.api.com",
		Model:       "custom-model",
		Temperature: 0.5,
		NumPredict:  2048,
		Theme:       "light",
	}
	data, _ := json.MarshalIndent(fileConfig, "", "  ")
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if cfg.APIKey != "file-api-key" {
		t.Errorf("cfg.APIKey = %q, want %q", cfg.APIKey, "file-api-key")
	}
	if cfg.BaseURL != "https://custom.api.com" {
		t.Errorf("cfg.BaseURL = %q, want %q", cfg.BaseURL, "https://custom.api.com")
	}
	if cfg.Model != "custom-model" {
		t.Errorf("cfg.Model = %q, want %q", cfg.Model, "custom-model")
	}
	if cfg.Temperature != 0.5 {
		t.Errorf("cfg.Temperature = %f, want %f", cfg.Temperature, 0.5)
	}
}

func TestSaveExcludesAPIKey(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, ".agenttea")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		APIKey:      "secret-key-should-not-be-saved",
		BaseURL:     "https://ollama.com",
		Model:       "qwen3.5:cloud",
		Temperature: 0.7,
		NumPredict:  4096,
		Theme:       "dark",
	}

	saveCfg := *cfg
	saveCfg.APIKey = ""

	data, err := json.MarshalIndent(saveCfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	var saved Config
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}

	if saved.APIKey != "" {
		t.Errorf("saved.APIKey = %q, want empty string (API key should not be persisted)", saved.APIKey)
	}
	if saved.Model != "qwen3.5:cloud" {
		t.Errorf("saved.Model = %q, want %q", saved.Model, "qwen3.5:cloud")
	}
}

func TestSaveFilePermissions(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("文件权限测试在 Windows 上不适用")
	}

	dir := t.TempDir()
	configDir := filepath.Join(dir, ".agenttea")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		BaseURL:     "https://ollama.com",
		Model:       "qwen3.5:cloud",
		Temperature: 0.7,
		NumPredict:  4096,
		Theme:       "dark",
	}

	saveCfg := *cfg
	saveCfg.APIKey = ""
	data, _ := json.MarshalIndent(saveCfg, "", "  ")
	configPath := filepath.Join(configDir, "config.json")

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatal(err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("config file permissions = %o, want %o", perm, 0600)
	}
}

func TestLoadFillsDefaults(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, ".agenttea")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	partialConfig := map[string]interface{}{
		"model": "gpt-oss:120b",
	}
	data, _ := json.MarshalIndent(partialConfig, "", "  ")
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if cfg.Model != "gpt-oss:120b" {
		t.Errorf("cfg.Model = %q, want %q", cfg.Model, "gpt-oss:120b")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultConfig.BaseURL
	}
	if cfg.BaseURL != DefaultConfig.BaseURL {
		t.Errorf("cfg.BaseURL should default to %q, got %q", DefaultConfig.BaseURL, cfg.BaseURL)
	}
	if cfg.Temperature == 0 {
		cfg.Temperature = DefaultConfig.Temperature
	}
	if cfg.Temperature != DefaultConfig.Temperature {
		t.Errorf("cfg.Temperature should default to %f, got %f", DefaultConfig.Temperature, cfg.Temperature)
	}
}

func TestAPIKeyOmitEmpty(t *testing.T) {
	cfg := Config{
		BaseURL: "https://ollama.com",
		Model:   "qwen3.5:cloud",
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}

	if _, exists := m["api_key"]; exists {
		t.Error("api_key should be omitted when empty (omitempty tag)")
	}
}
