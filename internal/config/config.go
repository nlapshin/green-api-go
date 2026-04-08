package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Host            string        `env:"APP_HOST" envDefault:"0.0.0.0"`
	Port            int           `env:"APP_PORT" envDefault:"8080"`
	WebRoot         string        `env:"WEB_ROOT" envDefault:"."`
	GreenAPIBaseURL string        `env:"GREEN_API_BASE_URL" envDefault:"https://api.green-api.com"`
	GreenAPITimeout time.Duration `env:"GREEN_API_TIMEOUT" envDefault:"15s"`
}

func Load() (Config, error) {
	var c Config
	if err := env.Parse(&c); err != nil {
		return Config{}, fmt.Errorf("config env: %w", err)
	}
	// PaaS convention (e.g. Render, Railway): use PORT when APP_PORT is empty or unset.
	if strings.TrimSpace(os.Getenv("APP_PORT")) == "" {
		if ps := strings.TrimSpace(os.Getenv("PORT")); ps != "" {
			p, err := strconv.Atoi(ps)
			if err != nil {
				return Config{}, fmt.Errorf("PORT: %w", err)
			}
			c.Port = p
		}
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

func (c Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("APP_PORT must be between 1 and 65535, got %d", c.Port)
	}
	if c.GreenAPITimeout <= 0 {
		return fmt.Errorf("GREEN_API_TIMEOUT must be positive, got %s", c.GreenAPITimeout)
	}
	return nil
}

func (c Config) IndexTemplatePath() string {
	return filepath.Join(c.WebRoot, "web", "templates", "index.html")
}

func (c Config) EnsureIndexTemplate() error {
	p := c.IndexTemplatePath()
	fi, err := os.Stat(p)
	if err != nil || fi.IsDir() {
		return fmt.Errorf("SSR template not found at %q (set WEB_ROOT to project root)", p)
	}
	return nil
}

func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c Config) StaticDir() string {
	return filepath.Join(c.WebRoot, "web", "static")
}
