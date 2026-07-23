package aggregator

import (
	"context"
	"flight-search-aggr-system/internal/domain"
	"flight-search-aggr-system/internal/provider"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
)

// ProviderSet records the outcome of one provider call
type ProviderSet struct {
	Provider  string
	LatencyMs int64
	Success   bool
}

// Result holds the aggregated output from all providers.
type Result struct {
	Flights        []domain.Flight
	ProviderErrors []error
	Stats          []ProviderSet
}

// Aggregator fans out search request to all registered adapters in parallel.
type Aggregator struct {
	registry           *provider.Registry
	perProviderTimeout time.Duration
}

// New creates an Aggregator.
func New(r *provider.Registry, timeout time.Duration) *Aggregator {
	return &Aggregator{
		registry:           r,
		perProviderTimeout: timeout,
	}
}

// Search fans out to all adapters concurrently and collects the results.
func (a *Aggregator) Search(ctx context.Context, req domain.SearchRequest) (Result, error) {
	adapters := a.registry.Adapters()
	results := make([][]domain.Flight, len(adapters))
	errs := make([]error, len(adapters))
	stats := make([]ProviderSet, len(adapters))

	var g errgroup.Group
	for i, ad := range adapters {
		i, ad := i, ad
		g.Go(func() error {
			start := time.Now()
			pctx, cancel := context.WithTimeout(ctx, a.perProviderTimeout)
			defer cancel()

			flights, err := ad.Fetch(pctx, req)
			results[i] = flights
			errs[i] = err
			stats[i] = ProviderSet{
				Provider:  ad.Name(),
				LatencyMs: time.Since(start).Milliseconds(),
				Success:   err == nil,
			}
			return nil
		})
	}
	g.Wait()

	var flights []domain.Flight
	var failed []error
	for i, fs := range results {
		if errs[i] != nil {
			failed = append(failed, errs[i])
			continue
		}
		flights = append(flights, fs...)
	}

	if len(failed) == len(adapters) {
		return Result{}, fmt.Errorf("%w: %v", provider.ErrAllProvidersFailed, failed)
	}

	return Result{
		Flights:        flights,
		ProviderErrors: failed,
		Stats:          stats,
	}, nil
}
