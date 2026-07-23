// Package simulator wraps a provider.Adapter with configurable latency and failure injection.
package simulator

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"flight-search-aggr-system/internal/domain"
	"flight-search-aggr-system/internal/provider"
)

// Simulated wraps an Adapter and adds delay and probabilistic failure.
type Simulated struct {
	inner       provider.Adapter
	minDelay    time.Duration
	maxDelay    time.Duration
	successRate float64

	mu  sync.Mutex
	rnd *rand.Rand
}

// New creates a Simulated decorator.
// seed makes failure/delay sequences reproducible in tests.
func New(inner provider.Adapter, minDelay, maxDelay time.Duration, successRate float64, seed int64) *Simulated {
	return &Simulated{
		inner:       inner,
		minDelay:    minDelay,
		maxDelay:    maxDelay,
		successRate: successRate,
		rnd:         rand.New(rand.NewSource(seed)),
	}
}

// Name delegates to the inner adapter.
func (s *Simulated) Name() string { return s.inner.Name() }

// Fetch injects delay and optional failure before calling the inner adapter.
func (s *Simulated) Fetch(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error) {
	s.mu.Lock()
	delay := s.randomDelay()
	fail := s.rnd.Float64() >= s.successRate
	s.mu.Unlock()

	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return nil, &provider.ProviderError{
			Provider: s.Name(),
			Kind:     provider.KindTimeout,
			Err:      ctx.Err(),
		}
	}

	if fail {
		return nil, &provider.ProviderError{
			Provider: s.Name(),
			Kind:     provider.KindTransport,
			Err:      errors.New("simulated transport failure"),
		}
	}

	return s.inner.Fetch(ctx, req)
}

func (s *Simulated) randomDelay() time.Duration {
	diff := s.maxDelay - s.minDelay
	if diff <= 0 {
		return s.minDelay
	}
	return s.minDelay + time.Duration(s.rnd.Int63n(int64(diff)))
}
