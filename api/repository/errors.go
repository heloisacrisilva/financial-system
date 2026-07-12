package repository

import "errors"

// Validation account errors
var (
	ErrAccountNotFound  = errors.New("account not found")
	ErrAccountNotActive = errors.New("account is not active")
	ErrInvalidCurrency  = errors.New("currency mismatch")
	ErrInvalidValue     = errors.New("value must be greater than zero")
)

// Credit/balance errors
var (
	ErrInsufficientFunds   = errors.New("insufficient funds including credit limit")
	ErrCreditLimitExceeded = errors.New("credit operation exceeds the allowed limit")
)

// Idempotency errors
var (
	ErrDuplicateRef = errors.New("reference_id already processed")
)
