package transaction

import "errors"

var (
	ErrInvalidUserID        = errors.New("invalid user ID")
	ErrInvalidTransactionID = errors.New("invalid transaction ID")
	ErrNotFound             = errors.New("transaction not found")
	ErrAccountNotFound      = errors.New("account not found")
	ErrAccountArchived      = errors.New("account is archived")
)
