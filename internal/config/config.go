// Package config loads and validates the service configuration from
// environment variables.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
)

// Config is the validated service configuration.
type Config struct {
	Port     int
	LogLevel slog.Level
}

// Load reads the configuration through getenv (os.Getenv in production) and
// reports every invalid or missing value at once.
func Load(getenv func(string) string) (Config, error) {
	var (
		cfg  Config
		errs []error
	)

	port, err := intVar(getenv, "PORT", 8080, 1, 65535)
	errs = append(errs, err)
	cfg.Port = port

	if v := getenv("LOG_LEVEL"); v != "" {
		if err := cfg.LogLevel.UnmarshalText([]byte(v)); err != nil {
			errs = append(errs, fmt.Errorf("LOG_LEVEL: %w", err))
		}
	}

	if err := errors.Join(errs...); err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}
	return cfg, nil
}

func intVar(getenv func(string) string, name string, def, lo, hi int) (int, error) {
	v := getenv(name)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s: not an integer: %q", name, v)
	}
	if n < lo || n > hi {
		return 0, fmt.Errorf("%s: %d out of range [%d, %d]", name, n, lo, hi)
	}
	return n, nil
}
