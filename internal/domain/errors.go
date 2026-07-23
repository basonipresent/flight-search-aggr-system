package domain

import "errors"

var (
	ErrInvalidRequest     = errors.New("invalid search request")
	ErrAllProvidersFailed = errors.New("all providers failed")
)

// ValidationError names the field that violated an invariant.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return "validation failed on " + e.Field + ": " + e.Reason
}
