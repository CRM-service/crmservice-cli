package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	API    APIConfig    `yaml:"api"`
	Output OutputConfig `yaml:"output"`
	Cache  CacheConfig  `yaml:"cache"`
	Auth   AuthConfig   `yaml:"auth"`
}

type APIConfig struct {
	URL     string `yaml:"url"`
	Timeout int    `yaml:"timeout"`
}

type OutputConfig struct {
	Format   string `yaml:"format"`
	PageSize int    `yaml:"page_size"`
}

type CacheConfig struct {
	SchemaDir   string `yaml:"schema_dir"`
	TTLDays     int    `yaml:"ttl_days"`
	AutoRefresh bool   `yaml:"auto_refresh"`
}

type AuthConfig struct {
	Token string `yaml:"token"`
	Type  string `yaml:"type"`
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

func defaultConfig() *Config {
	return &Config{
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
}

func LoadConfig(configFile string) (*Config, error) {
	config := defaultConfig()

	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, err
		}
	}

	applyEnv(config)

	return config, nil
}

func applyEnv(config *Config) {
	if value := os.Getenv("CRMSERVICE_API_URL"); value != "" {
		config.API.URL = value
	}
	if value := os.Getenv("CRMSERVICE_AUTH_TOKEN"); value != "" {
		config.Auth.Token = value
	}
	if value := os.Getenv("CRMSERVICE_OUTPUT_FORMAT"); value != "" {
		config.Output.Format = value
	}
	if value := os.Getenv("CRMSERVICE_PAGE_SIZE"); value != "" {
		if pageSize, err := strconv.Atoi(value); err == nil {
			config.Output.PageSize = pageSize
		}
	}
	if value := os.Getenv("CRMSERVICE_TIMEOUT"); value != "" {
		if timeout, err := strconv.Atoi(value); err == nil {
			config.API.Timeout = timeout
		}
	}
	if value := os.Getenv("CRMSERVICE_CACHE_DIR"); value != "" {
		config.Cache.SchemaDir = value
	}
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
