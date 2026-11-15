package config

import (
	"fmt"

	platformconfig "github.com/marco-scarnato/multiservices-backend/libs/platform/config"
)

// Load reads and validates the technical configuration for the ckguesser-be service.
func Load() (platformconfig.Config, error) {
	cfg, err := platformconfig.Load(platformconfig.LoadOptions{
		DefaultServiceName: "ckguesser-be",
		DefaultHTTPAddr:    ":8081",
	})
	if err != nil {
		return platformconfig.Config{}, err
	}
	if cfg.ServiceName != "ckguesser-be" {
		return platformconfig.Config{}, fmt.Errorf("SERVICE_NAME must be ckguesser-be, got %q", cfg.ServiceName)
	}
	return cfg, nil
}
