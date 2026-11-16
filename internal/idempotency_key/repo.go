package idempotency_key

import "context"

type TxRunnerRepo interface {
	Run(ctx context.Context, fn func(ctx context.Context) error) error
}

type IdempotencyKeyRepo interface {
	Claim(ctx context.Context, scope Scope, fingerprint []byte) (existing *Record, err error)
	Complete(ctx context.Context, scope Scope, result Result) error
}
