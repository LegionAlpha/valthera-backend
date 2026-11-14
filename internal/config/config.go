package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr            string
	DatabaseURL         string
	DevUserID           string
	RegistrationEnabled bool
}

func Load() (Config, error) {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}

	var registrationEnabled bool

	if raw := os.Getenv("REGISTRATION_ENABLED"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf(
				"parse REGISTRATION_ENABLED: %w",
				err,
			)
		}
		registrationEnabled = value
	}

	return Config{
		HTTPAddr:            addr,
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		DevUserID:           os.Getenv("DEV_USER_ID"),
		RegistrationEnabled: registrationEnabled,
	}, nil
}
