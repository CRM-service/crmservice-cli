package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	API    APIConfig    `mapstructure:"api"`
	Output OutputConfig `mapstructure:"output"`
	Cache  CacheConfig  `mapstructure:"cache"`
	Auth   AuthConfig   `mapstructure:"auth"`
}

type APIConfig struct {
	URL     string `mapstructure:"url"`
	Timeout int    `mapstructure:"timeout"`
}

type OutputConfig struct {
	Format   string `mapstructure:"format"`
	PageSize int    `mapstructure:"page_size"`
}

type CacheConfig struct {
	SchemaDir   string `mapstructure:"schema_dir"`
	TTLDays     int    `mapstructure:"ttl_days"`
	AutoRefresh bool   `mapstructure:"auto_refresh"`
}

type AuthConfig struct {
	Token string `mapstructure:"token"`
	Type  string `mapstructure:"type"`
}

func addAPISuffix(url *string) {
	if !strings.HasSuffix(*url, "/api/v1") {
		*url = strings.TrimSuffix(*url, "/") + "/api/v1"
	}
}

func sanitizeURL(url string) string {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "https://" + url
	}
	if strings.HasPrefix(url, "http://") {
		return "https://" + strings.TrimPrefix(url, "http://")
	}
	return url
}

func LoadConfig(configFile string) (*Config, error) {
	v := viper.New()

	if configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			return nil, err
		}
	}

	v.SetEnvPrefix("CRMSERVICE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	config := &Config{
		API: APIConfig{
			Timeout: 30,
		},
		Output: OutputConfig{
			Format:   "table",
			PageSize: 20,
		},
		Cache: CacheConfig{
			SchemaDir:   getCacheDir(),
			TTLDays:     24,
			AutoRefresh: true,
		},
		Auth: AuthConfig{
			Type: "bearer",
		},
	}

	if err := v.Unmarshal(config); err != nil {
		return nil, err
	}

	return config, nil
}

func getCacheDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".cache", "crmservice", "schema")
	}
	return filepath.Join(os.TempDir(), "crmservice", "schema")
}

func (c *Config) GetAPIURL() string {
	return strings.TrimSuffix(c.API.URL, "/")
}

func (c *Config) GetSanitizedAPIURL() string {
	url := c.API.URL

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	if strings.HasPrefix(url, "http://") {
		url = "https://" + strings.TrimPrefix(url, "http://")
	}

	return strings.TrimSuffix(url, "/")
}
