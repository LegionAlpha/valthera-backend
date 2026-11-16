package user_session

import (
	"context"

	"github.com/google/uuid"
)

type UserSessionService struct {
	repo UserSessionRepo
}

func NewUserSessionService(repo UserSessionRepo) *UserSessionService {
	return &UserSessionService{
		repo: repo,
	}
}

func (s *UserSessionService) GetOrCreate(ctx context.Context, user_id uuid.UUID) (UserSession, error) {

}
