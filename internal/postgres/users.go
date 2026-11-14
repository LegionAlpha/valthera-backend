package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/LegionAlpha/valthera-backend/internal/identity"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		pool: pool,
	}
}

var _ identity.UserCreator = (*UserRepository)(nil)

func (r *UserRepository) Create(
	ctx context.Context,
	params identity.CreateUserParams,
) (identity.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var created identity.User

	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (
			email,
			password_hash
		)
		VALUES ($1, $2)
		RETURNING
			id,
			email,
			created_at
	`,
		params.Email,
		params.PasswordHash,
	).Scan(
		&created.ID,
		&created.Email,
		&created.CreatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) &&
			pgErr.Code == "23505" &&
			pgErr.ConstraintName == "users_email_unique" {
			return identity.User{}, identity.ErrEmailAlreadyExists
		}

		return identity.User{}, fmt.Errorf("insert user: %w", err)
	}

	return created, nil
}
