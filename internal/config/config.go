package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultConfigDirName  = ".nest-cli"
	defaultConfigFileName = "config.yaml"
	ModeMongo             = "mongo"
	ModeAPI               = "api"
)

type Origin struct {
	Mode          string `yaml:"mode,omitempty"`
	APIBaseURL    string `yaml:"api_base_url,omitempty"`
	MongoURI      string `yaml:"mongo_uri,omitempty"`
	MongoDatabase string `yaml:"mongo_database,omitempty"`
	TLSPinSHA     string `yaml:"tls_pin_sha256,omitempty"`
}

func (o Origin) EffectiveMode() string {
	if o.Mode == ModeMongo || o.Mode == ModeAPI {
		return o.Mode
	}

	// Backward compatibility: old configs with api_base_url and no mode stay in API mode.
	if o.APIBaseURL != "" {
		return ModeAPI
	}

	return ModeMongo
}

func ResolveMongoURI(originName string, o Origin) string {
	// Preserve explicit config preference when provided.
	if strings.TrimSpace(o.MongoURI) != "" {
		return strings.TrimSpace(o.MongoURI)
	}

	if originEnv := os.Getenv("NEST_MONGO_URI_" + NormalizeOriginEnvKey(originName)); strings.TrimSpace(originEnv) != "" {
		return strings.TrimSpace(originEnv)
	}

	if globalEnv := os.Getenv("NEST_MONGO_URI"); strings.TrimSpace(globalEnv) != "" {
		return strings.TrimSpace(globalEnv)
	}

	return ""
}

func NormalizeOriginEnvKey(originName string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(originName) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

type Config struct {
	DefaultOrigin string            `yaml:"default_origin"`
	CryptoProfile string            `yaml:"crypto_profile"`
	ActiveKeyID   string            `yaml:"active_key_id"`
	AuthToken     string            `yaml:"auth_token,omitempty"`
	Origins       map[string]Origin `yaml:"origins"`
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}

	return filepath.Join(home, defaultConfigDirName, defaultConfigFileName), nil
}

func LoadOrCreate(path string) (*Config, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		cfg := Default()
		if err := Save(path, cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	return Load(path)
}

func Default() *Config {
	return &Config{
		CryptoProfile: "modern",
		Origins:       map[string]Origin{},
	}
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %q: %w", path, err)
	}

	cfg := Default()
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, fmt.Errorf("decode config file %q: %w", path, err)
	}

	if cfg.Origins == nil {
		cfg.Origins = map[string]Origin{}
	}

	return cfg, nil
}

func Save(path string, cfg *Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config directory %q: %w", dir, err)
	}

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	if err := os.WriteFile(path, b, 0o600); err != nil {
		return fmt.Errorf("write config file %q: %w", path, err)
	}

	return nil
}
