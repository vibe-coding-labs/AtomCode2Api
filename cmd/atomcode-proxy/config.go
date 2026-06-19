package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:     "config",
	Short:   "管理配置",
	Long:    "查看和修改 AtomCode Proxy 配置。",
	GroupID: "core",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "显示当前配置",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		data, _ := json.MarshalIndent(cfg, "", "  ")
		fmt.Println(string(data))
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}

func configPath() string {
	if p := os.Getenv("ATOMCODE_PROXY_CONFIG"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".atomcode-proxy", "config.json")
}

type Config struct {
	Server   ServerConfig   `json:"server"`
	Daemon   DaemonConfig   `json:"daemon"`
	Logging  LoggingConfig  `json:"logging"`
	Sessions SessionsConfig `json:"sessions"`
}

type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	TLS  bool   `json:"tls"`
}

type DaemonConfig struct {
	URL        string `json:"url"`
	AutoManage bool   `json:"auto_manage"`
	Telemetry  bool   `json:"telemetry"`
}

type LoggingConfig struct {
	Level string `json:"level"`
	File  string `json:"file,omitempty"`
}

type SessionsConfig struct {
	MaxAgeMinutes int `json:"max_age_minutes"`
}

func defaultConfig() Config {
	return Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 13457,
			TLS:  false,
		},
		Daemon: DaemonConfig{
			URL:        getEnvDefault("ATOMCODE_DAEMON_URL", "http://localhost:13456"),
			AutoManage: false,
			Telemetry:  true,
		},
		Logging: LoggingConfig{
			Level: "info",
		},
		Sessions: SessionsConfig{
			MaxAgeMinutes: 30,
		},
	}
}

func loadConfig() Config {
	cfg := defaultConfig()
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	var loaded Config
	json.Unmarshal(data, &loaded)
	cfg.Server.Host = firstNonEmpty(loaded.Server.Host, cfg.Server.Host)
	if loaded.Server.Port > 0 {
		cfg.Server.Port = loaded.Server.Port
	}
	cfg.Daemon.URL = firstNonEmpty(loaded.Daemon.URL, cfg.Daemon.URL)
	return cfg
}

func saveConfig(cfg Config) error {
	dir := filepath.Dir(configPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(configPath(), data, 0644)
}

func firstNonEmpty(s, fallback string) string {
	if s != "" {
		return s
	}
	return fallback
}
