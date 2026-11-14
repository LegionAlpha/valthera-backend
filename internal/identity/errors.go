package identity

import "errors"

var ErrInvalidEmail = errors.New("invalid email")
var ErrInvalidPassword = errors.New("invalid password")
var ErrPasswordHashingBusy = errors.New("password hashing capacity exhausted")
var ErrEmailAlreadyExists = errors.New("email already exists")
