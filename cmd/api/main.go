package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/LegionAlpha/valthera-backend/internal/account"
	"github.com/LegionAlpha/valthera-backend/internal/config"
	"github.com/LegionAlpha/valthera-backend/internal/httpapi"
	"github.com/LegionAlpha/valthera-backend/internal/identity"
	"github.com/LegionAlpha/valthera-backend/internal/password"
	"github.com/LegionAlpha/valthera-backend/internal/postgres"
	"github.com/LegionAlpha/valthera-backend/internal/transaction"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// .env is optional: in Docker the variables come from the environment.
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	var devUserID uuid.UUID

	if cfg.DevUserID != "" {
		parsedID, err := uuid.Parse(cfg.DevUserID)
		if err != nil {
			return fmt.Errorf("parse DEV_USER_ID: %w", err)
		}
		if parsedID == uuid.Nil {
			return errors.New("DEV_USER_ID must not be a nil UUID")
		}

		devUserID = parsedID
	}

	if cfg.DevUserID != "" || cfg.RegistrationEnabled {
		host, _, err := net.SplitHostPort(cfg.HTTPAddr)
		if err != nil {
			return fmt.Errorf("parse HTTP_ADDR: %w", err)
		}

		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return errors.New(
				"development identity and registration require a loopback HTTP_ADDR",
			)
		}
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

	if devUserID != uuid.Nil {
		var exists bool

		err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM users
			WHERE id = $1
		)
	`, devUserID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("validate DEV_USER_ID: %w", err)
		}

		if !exists {
			return errors.New(
				"DEV_USER_ID does not reference an existing user",
			)
		}

		log.Println("local development identity enabled")
	}

	cancel()

	accountRepository := postgres.NewAccountRepository(pool)
	accountService := account.NewService(
		accountRepository,
		accountRepository,
	)

	transactionRepository := postgres.NewTransactionRepository(pool)
	transactionService := transaction.NewService(
		transactionRepository,
		transactionRepository,
	)

	passwordHasher, err := password.NewArgon2idHasher(2)
	if err != nil {
		return fmt.Errorf("create password hasher: %w", err)
	}

	userRepository := postgres.NewUserRepository(pool)
	identityService := identity.NewService(
		userRepository,
		passwordHasher,
	)

	handler := httpapi.NewHandler(
		accountService,
		transactionService,
		identityService,
		devUserID,
		"http://"+cfg.HTTPAddr,
		cfg.RegistrationEnabled,
	)

	log.Println("database connected")

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("starting server on http://%s", server.Addr)

	return server.ListenAndServe()
}
