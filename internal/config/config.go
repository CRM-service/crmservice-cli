package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"crmservice/internal/cache"
	"crmservice/internal/output"

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
	TTLSeconds  int    `yaml:"ttl_seconds"`
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
			TTLSeconds:  cache.DefaultTTLSeconds,
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

	if err := applyEnv(config); err != nil {
		return nil, err
	}
	if err := validateConfig(config); err != nil {
		return nil, err
	}
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

func applyEnv(config *Config) error {
	if value := os.Getenv("CRMSERVICE_API_URL"); value != "" {
		config.API.URL = value
	}
	if value := os.Getenv("CRMSERVICE_AUTH_TOKEN"); value != "" {
		config.Auth.Token = value
	}
	if value := os.Getenv("CRMSERVICE_OUTPUT_FORMAT"); value != "" {
		if !output.ValidOutputFormat(value) {
			return fmt.Errorf("invalid CRMSERVICE_OUTPUT_FORMAT: %q", value)
		}
		config.Output.Format = value
	}
	if value := os.Getenv("CRMSERVICE_PAGE_SIZE"); value != "" {
		pageSize, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid CRMSERVICE_PAGE_SIZE: %q", value)
		}
		if err := validatePageSize(pageSize, "CRMSERVICE_PAGE_SIZE"); err != nil {
			return err
		}
		config.Output.PageSize = pageSize
	}
	if value := os.Getenv("CRMSERVICE_TIMEOUT"); value != "" {
		timeout, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid CRMSERVICE_TIMEOUT: %q", value)
		}
		config.API.Timeout = timeout
	}
	if value := os.Getenv("CRMSERVICE_CACHE_DIR"); value != "" {
		config.Cache.SchemaDir = value
	}
	if value := os.Getenv("CRMSERVICE_CACHE_TTL_SECONDS"); value != "" {
		ttlSeconds, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid CRMSERVICE_CACHE_TTL_SECONDS: %q", value)
		}
		config.Cache.TTLSeconds = ttlSeconds
	}
	if value := os.Getenv("CRMSERVICE_CACHE_AUTO_REFRESH"); value != "" {
		autoRefresh, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid CRMSERVICE_CACHE_AUTO_REFRESH: %q", value)
		}
		config.Cache.AutoRefresh = autoRefresh
	}
	return nil
}

func validateConfig(config *Config) error {
	if err := validatePageSize(config.Output.PageSize, "output.page_size"); err != nil {
		return err
	}
	return nil
}

func validatePageSize(pageSize int, source string) error {
	if pageSize < 1 {
		return fmt.Errorf("invalid %s: %d (must be at least 1)", source, pageSize)
	}
	return nil
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
	return DisplayAPIURL(c.API.URL)
}

func (c *Config) GetSanitizedAPIURL() (string, error) {
	return ResolveAPIURL(c.API.URL, false)
}
