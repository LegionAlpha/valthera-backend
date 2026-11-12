package config

import "os"

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	DevUserID   string
}

func Load() Config {
	addr := os.Getenv("HTTP_ADDR")

	if addr == "" {
		addr = "127.0.0.1:8080"
	}

	return Config{
		HTTPAddr:    addr,
		DatabaseURL: os.Getenv("DATABASE_URL"),
		DevUserID:   os.Getenv("DEV_USER_ID"),
	}
}
