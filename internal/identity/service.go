package identity

import (
	"context"
	"fmt"
)

type RegisterInput struct {
	Email    string
	Password string
}

type CreateUserParams struct {
	Email        string
	PasswordHash string
}

type UserCreator interface {
	Create(
		ctx context.Context,
		params CreateUserParams,
	) (User, error)
}

type Service struct {
	users  UserCreator
	hasher PasswordHasher
}

func NewService(
	users UserCreator,
	hasher PasswordHasher,
) *Service {
	return &Service{
		users:  users,
		hasher: hasher,
	}
}

func (s *Service) Register(
	ctx context.Context,
	input RegisterInput,
) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}

	email, err := NormalizeEmail(input.Email)
	if err != nil {
		return User{}, err
	}

	if err := ValidatePassword(input.Password); err != nil {
		return User{}, err
	}

	passwordHash, err := s.hasher.Hash(ctx, input.Password)
	if err != nil {
		return User{}, fmt.Errorf("hash registration password: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return User{}, err
	}

	created, err := s.users.Create(ctx, CreateUserParams{
		Email:        email,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return User{}, fmt.Errorf("register user: %w", err)
	}

	return created, nil
}
