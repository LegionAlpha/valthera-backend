package idempotency_key

import (
	"bytes"
	"context"
)

type Service struct {
	repo     IdempotencyKeyRepo
	txRunner TxRunnerRepo
}

func NewService(repository IdempotencyKeyRepo, txRunner TxRunnerRepo) *Service {
	return &Service{
		repo:     repository,
		txRunner: txRunner,
	}
}

func (s *Service) Do(ctx context.Context, scope Scope, fingerprint []byte, work func(ctx context.Context) (Result, error)) (Result, error) {
	var result Result

	err := s.txRunner.Run(ctx, func(ctx context.Context) error {
		existing, err := s.repo.Claim(ctx, scope, fingerprint)
		if err != nil {
			return err
		}

		if existing != nil {
			if !bytes.Equal(existing.Fingerprint, fingerprint) {
				return ErrKeyReused
			}
			result = existing.Result
			return nil
		}

		res, err := work(ctx)
		if err != nil {
			return err
		}

		if err := s.repo.Complete(ctx, scope, res); err != nil {
			return err
		}

		result = res
		return nil
	})
	if err != nil {
		return Result{}, err
	}

	return result, nil

}
