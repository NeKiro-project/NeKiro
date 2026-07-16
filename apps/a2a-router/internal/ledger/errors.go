package ledger

import "errors"

var (
	ErrConflict   = errors.New("invocation ledger conflict")
	ErrDependency = errors.New("invocation ledger dependency failure")
	ErrNotFound   = errors.New("invocation ledger fact not found")
	ErrValidation = errors.New("invocation ledger validation failed")
)
