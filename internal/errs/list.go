package errs

import "errors"

var (
	ErrInvalidRequestBody    = errors.New("invalid request body")
	ErrInvalidFieldValue     = errors.New("invalid field value")
	ErrNotFound              = errors.New("not found")
	ErrContractNotFound      = errors.New("contract not found")
	ErrAccessDenied          = errors.New("access denied")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrContractAlreadyExists = errors.New("contract already exists")
	ErrPaymentExceedsBalance = errors.New("payment exceeds contract balance, additional agreement required")
)