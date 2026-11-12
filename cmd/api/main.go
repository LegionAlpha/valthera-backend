package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/LegionAlpha/valthera-backend/internal/config"
	"github.com/LegionAlpha/valthera-backend/internal/httpapi"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.Load()

	if cfg.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	cancel()

	log.Println("database connected")

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewHandler(),
		ReadHeaderTimeout: 3 * time.Second,
	}

	log.Printf("starting server on http://%s", server.Addr)

	return server.ListenAndServe()
}
