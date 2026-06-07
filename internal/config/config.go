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
		Auth: AuthConfig{},
	}
}

func LoadConfig(configFile string) (*Config, error) {
	config := defaultConfig()

	configPath := configFile
	usingDefaultPath := false
	if configPath == "" {
		configPath = defaultConfigFile()
		usingDefaultPath = true
	}

	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			if !(usingDefaultPath && os.IsNotExist(err)) {
				return nil, err
			}
		} else if err := yaml.Unmarshal(data, config); err != nil {
			return nil, err
		}
	}

	applyEnv(config)
	ExpandPaths(config)

	return config, nil
}

var userConfigDir = os.UserConfigDir
var userCacheDir = os.UserCacheDir
var userHomeDir = os.UserHomeDir

func defaultConfigFile() string {
	configDir, err := userConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(configDir, "crmservice", "config.yaml")
}

func DefaultConfigPath() string {
	return defaultConfigFile()
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
	if value := os.Getenv("CRMSERVICE_CACHE_TTL_DAYS"); value != "" {
		if ttlDays, err := strconv.Atoi(value); err == nil {
			config.Cache.TTLDays = ttlDays
		}
	}
	if value := os.Getenv("CRMSERVICE_CACHE_AUTO_REFRESH"); value != "" {
		if autoRefresh, err := strconv.ParseBool(value); err == nil {
			config.Cache.AutoRefresh = autoRefresh
		}
	}
}

// ExpandPaths resolves home-directory shortcuts in config paths.
func ExpandPaths(config *Config) {
	config.Cache.SchemaDir = expandHomePath(config.Cache.SchemaDir)
}

func expandHomePath(path string) string {
	if path == "~" {
		home, err := userHomeDir()
		if err != nil {
			return path
		}
		return home
	}
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		home, err := userHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func getCacheDir() string {
	cacheDir, err := userCacheDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "crmservice", "schema")
	}
	return filepath.Join(cacheDir, "crmservice", "schema")
}

func (c *Config) GetAPIURL() string {
	return strings.TrimSuffix(c.API.URL, "/")
}

func (c *Config) GetSanitizedAPIURL() string {
	url := c.API.URL

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	return strings.TrimSuffix(url, "/")
}
