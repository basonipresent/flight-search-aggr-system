// Package provider defines the Adapter interface and shared error types.
package provider

import (
	"context"
	"errors"
	"fmt"

	"flight-search-aggr-system/internal/domain"
)

// Adapter is implemented by each airline-specific adapter.
type Adapter interface {
	Name() string
	Fetch(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error)
}

// ErrorKind classifies a provider failure for retry decisions.
// timeout and transport are retryable; decode and validation are not.
type ErrorKind string

const (
	KindTimeout    ErrorKind = "timeout"
	KindTransport  ErrorKind = "transport"
	KindDecode     ErrorKind = "decode"
	KindValidation ErrorKind = "validation"
)

// ProviderError wraps a provider failure with its source and classification.
type ProviderError struct {
	Provider string
	Kind     ErrorKind
	Err      error
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("provider %s [%s]: %v", e.Provider, e.Kind, e.Err)
}

func (e *ProviderError) Unwrap() error { return e.Err }

// ErrAllProvidersFailed is returned by the aggregator when every adapter fails.
var ErrAllProvidersFailed = errors.New("all providers failed")

// Registry holds the configured set of adapters the aggregator fans out to.
type Registry struct {
	adapters []Adapter
}

// NewRegistry creates a Registry from the given adapters.
func NewRegistry(adapters ...Adapter) *Registry {
	return &Registry{adapters: adapters}
}

// Adapters returns the registered adapters.
func (r *Registry) Adapters() []Adapter {
	return r.adapters
}
