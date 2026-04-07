package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	DefaultFile                 = "config.yaml"
	DefaultStrategy             = "round_robin"
	WeightedRoundRobinStrategy  = "weighted_round_robin"
	LeastConnectionsStrategy    = "least_connections"
	DefaultDialTimeout          = 3 * time.Second
	DefaultTLSHandshakeTimeout  = 3 * time.Second
	DefaultResponseHeaderTimout = 10 * time.Second
	DefaultHealthInterval       = 5 * time.Second
	DefaultHealthTimeout        = 2 * time.Second
)

var (
	ErrNoBackends        = errors.New("config requires at least one backend")
	ErrInvalidStrategy   = errors.New("config strategy is invalid")
	ErrInvalidBackendURL = errors.New("config backend url is required")
)

type Config struct {
	Port     string         `yaml:"port"`
	Strategy string         `yaml:"strategy"`
	Backends []Backend      `yaml:"backends"`
	Timeouts TimeoutConfig  `yaml:"timeouts"`
}

type Backend struct {
	URL    string `yaml:"url"`
	Weight int    `yaml:"weight"`
}

type TimeoutConfig struct {
	Dial           Duration `yaml:"dial"`
	TLSHandshake   Duration `yaml:"tls_handshake"`
	ResponseHeader Duration `yaml:"response_header"`
	HealthInterval Duration `yaml:"health_interval"`
	HealthTimeout  Duration `yaml:"health_timeout"`
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	var value string
	if err := node.Decode(&value); err != nil {
		return err
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", value, err)
	}

	d.Duration = parsed
	return nil
}

func (d Duration) MarshalYAML() (interface{}, error) {
	return d.Duration.String(), nil
}

func Load(path string) (Config, error) {
	if path == "" {
		path = DefaultFile
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	cfg.applyDefaults()

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func Save(path string, cfg Config) error {
	cfg.applyDefaults()

	if err := cfg.Validate(); err != nil {
		return err
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.Port) == "" {
		return errors.New("config port is required")
	}

	if len(c.Backends) == 0 {
		return ErrNoBackends
	}

	switch c.Strategy {
	case DefaultStrategy, WeightedRoundRobinStrategy, LeastConnectionsStrategy:
	default:
		return fmt.Errorf("%w: %s", ErrInvalidStrategy, c.Strategy)
	}

	for _, backend := range c.Backends {
		if strings.TrimSpace(backend.URL) == "" {
			return ErrInvalidBackendURL
		}

		if backend.Weight <= 0 {
			return errors.New("config backend weight must be greater than zero")
		}
	}

	if c.Timeouts.Dial.Duration <= 0 {
		return errors.New("config timeouts.dial must be greater than zero")
	}

	if c.Timeouts.TLSHandshake.Duration <= 0 {
		return errors.New("config timeouts.tls_handshake must be greater than zero")
	}

	if c.Timeouts.ResponseHeader.Duration <= 0 {
		return errors.New("config timeouts.response_header must be greater than zero")
	}

	if c.Timeouts.HealthInterval.Duration <= 0 {
		return errors.New("config timeouts.health_interval must be greater than zero")
	}

	if c.Timeouts.HealthTimeout.Duration <= 0 {
		return errors.New("config timeouts.health_timeout must be greater than zero")
	}

	return nil
}

func (c *Config) applyDefaults() {
	if strings.TrimSpace(c.Strategy) == "" {
		c.Strategy = DefaultStrategy
	}

	for i := range c.Backends {
		if c.Backends[i].Weight == 0 {
			c.Backends[i].Weight = 1
		}
	}

	if c.Timeouts.Dial.Duration == 0 {
		c.Timeouts.Dial.Duration = DefaultDialTimeout
	}

	if c.Timeouts.TLSHandshake.Duration == 0 {
		c.Timeouts.TLSHandshake.Duration = DefaultTLSHandshakeTimeout
	}

	if c.Timeouts.ResponseHeader.Duration == 0 {
		c.Timeouts.ResponseHeader.Duration = DefaultResponseHeaderTimout
	}

	if c.Timeouts.HealthInterval.Duration == 0 {
		c.Timeouts.HealthInterval.Duration = DefaultHealthInterval
	}

	if c.Timeouts.HealthTimeout.Duration == 0 {
		c.Timeouts.HealthTimeout.Duration = DefaultHealthTimeout
	}
}
