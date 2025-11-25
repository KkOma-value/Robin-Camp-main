package config

import (
	"fmt"
	"os"
	"strings"
)

// Config captures all runtime settings sourced from environment variables only.
type Config struct {
	Port         string
	AuthToken    string
	DatabaseURL  string
	BoxOfficeURL string
	BoxOfficeKey string
}

// Load reads required settings from the process environment and enforces presence.
func Load() (Config, error) {
	cfg := Config{
		Port:         strings.TrimSpace(os.Getenv("PORT")),
		AuthToken:    strings.TrimSpace(os.Getenv("AUTH_TOKEN")),
		DatabaseURL:  strings.TrimSpace(os.Getenv("DB_URL")),
		BoxOfficeURL: strings.TrimSpace(os.Getenv("BOXOFFICE_URL")),
		BoxOfficeKey: strings.TrimSpace(os.Getenv("BOXOFFICE_API_KEY")),
	}

	missing := cfg.missingFields()
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

// HTTPAddr returns a TCP address usable by net/http (e.g. 0.0.0.0:8080).
func (c Config) HTTPAddr() string {
	if strings.HasPrefix(c.Port, ":") {
		return "0.0.0.0" + c.Port
	}
	return "0.0.0.0:" + c.Port
}

func (c Config) missingFields() []string {
	var missing []string

	if c.Port == "" {
		missing = append(missing, "PORT")
	}
	if c.AuthToken == "" {
		missing = append(missing, "AUTH_TOKEN")
	}
	if c.DatabaseURL == "" {
		missing = append(missing, "DB_URL")
	}
	if c.BoxOfficeURL == "" {
		missing = append(missing, "BOXOFFICE_URL")
	}
	if c.BoxOfficeKey == "" {
		missing = append(missing, "BOXOFFICE_API_KEY")
	}

	return missing
}

// MustLoad wraps Load and panics; useful for tests/short-lived tools.
func MustLoad() Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}
