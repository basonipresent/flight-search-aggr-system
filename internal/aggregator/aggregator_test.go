package aggregator_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"flight-search-aggr-system/internal/aggregator"
	"flight-search-aggr-system/internal/domain"
	"flight-search-aggr-system/internal/provider"
)

type fakeAdapter struct {
	name    string
	flights []domain.Flight
	err     error
	delay   time.Duration
}

func (f *fakeAdapter) Name() string {
	return f.name
}

func (f *fakeAdapter) Fetch(ctx context.Context, _ domain.SearchRequest) ([]domain.Flight, error) {
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return f.flights, f.err
}

func makeFlights(n int) []domain.Flight {
	dep := time.Date(2025, 12, 15, 6, 0, 0, 0, time.UTC)
	arr := dep.Add(2 * time.Hour)
	flights := make([]domain.Flight, n)
	for i := range flights {
		f := domain.NewFlight()
		f.Price = domain.Money{Amount: 500_000, Currency: "IDR"}
		f.CabinClass = domain.Economy
		f.Departure = domain.Endpoint{Instant: dep}
		f.Arrival = domain.Endpoint{Instant: arr}
		f.Duration = domain.NewDuration(dep, arr)
		flights[i] = f
	}
	return flights
}

func TestAggregator_PartialFailure(t *testing.T) {
	reg := provider.NewRegistry(
		&fakeAdapter{name: "ok1", flights: makeFlights(2)},
		&fakeAdapter{name: "fail", err: errors.New("boom")},
		&fakeAdapter{name: "ok2", flights: makeFlights(3)},
	)
	agg := aggregator.New(reg, time.Second)

	res, err := agg.Search(context.Background(), domain.SearchRequest{})
	if err != nil {
		t.Fatalf("expected partial success, got error: %v", err)
	}
	if len(res.Flights) != 5 {
		t.Errorf("flights = %d, want 5", len(res.Flights))
	}
	failedCount := 0
	for _, e := range res.ProviderErrors {
		if e != nil {
			failedCount++
		}
	}
	if failedCount != 1 {
		t.Errorf("providers_failed = %d, want 1", failedCount)
	}
}

func TestAggregator_AllFail(t *testing.T) {
	reg := provider.NewRegistry(
		&fakeAdapter{name: "a", err: errors.New("err a")},
		&fakeAdapter{name: "b", err: errors.New("err b")},
	)
	agg := aggregator.New(reg, time.Second)

	_, err := agg.Search(context.Background(), domain.SearchRequest{})
	if err == nil {
		t.Fatal("expected error when all providers fail")
	}
	if !errors.Is(err, provider.ErrAllProvidersFailed) {
		t.Errorf("error = %v, want ErrAllProvidersFailed", err)
	}
}

func TestAggregator_ProviderTimeout(t *testing.T) {
	reg := provider.NewRegistry(
		&fakeAdapter{name: "fast", flights: makeFlights(1)},
		&fakeAdapter{name: "slow", delay: 500 * time.Millisecond},
	)
	agg := aggregator.New(reg, 100*time.Millisecond)

	start := time.Now()
	res, err := agg.Search(context.Background(), domain.SearchRequest{})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected partial success, got: %v", err)
	}
	if len(res.Flights) != 1 {
		t.Errorf("flights = %d, want 1 (slow provider timed out)", len(res.Flights))
	}
	if elapsed > 300*time.Millisecond {
		t.Errorf("elapsed %v, want < 300ms (should not wait for slow provider)", elapsed)
	}
}

func TestAggregator_ParentContextCancel(t *testing.T) {
	reg := provider.NewRegistry(
		&fakeAdapter{name: "slow", delay: time.Second},
	)
	agg := aggregator.New(reg, 5*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, _ = agg.Search(ctx, domain.SearchRequest{})
	if time.Since(start) > 500*time.Millisecond {
		t.Error("parent cancel did not abort providers in time")
	}
}
