package repository

import "errors"

// Validation account errors
var (
	ErrAccountNotFound  = errors.New("account not found")
	ErrAccountNotActive = errors.New("account is not active")
	ErrInvalidCurrency  = errors.New("currency mismatch")
	ErrInvalidValue     = errors.New("value must be greater than zero")
	ErrSameAccount      = errors.New("origin and destination accounts must be different")
)

// Credit/balance errors
var (
	ErrInsufficientFunds   = errors.New("insufficient funds including credit limit")
	ErrCreditLimitExceeded = errors.New("credit operation exceeds the allowed limit")
)

// Transaction errors
var (
	ErrTransactionNotFound      = errors.New("original transaction not found")
	ErrAlreadyReversed          = errors.New("transaction has already been reversed")
	ErrTransactionNotReversible = errors.New("transaction type or status does not support reversal")
)

// Reversal erros
var (
	ErrReversalValueMismatch = errors.New("reversal value must be the same of original value operation")
)

// Idempotency errors
var (
	ErrDuplicateRef = errors.New("reference_id already processed")
)
