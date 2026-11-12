package account

import "errors"

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrAggregateOutOfRange = errors.New(
		"aggregate is out of range",
	)
	ErrInvalidUserID = errors.New("invalid user ID")
)
