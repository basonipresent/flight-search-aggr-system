package cache_test

import (
	"flight-search-aggr-system/internal/cache"
	"flight-search-aggr-system/internal/domain"
	"sync"
	"testing"
	"time"
)

func baseReq() domain.SearchRequest {
	return domain.SearchRequest{
		Origin:      "CGK",
		Destination: "DPS",
		Passengers:  1,
		CabinClass:  domain.Economy,
	}
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

func TestCache_MissHit(t *testing.T) {
	c := cache.New(time.Minute)
	req := baseReq()

	if _, ok := c.Get(req); ok {
		t.Fatal("expected cache miss before Set")
	}
	c.Set(req, makeFlights(3))

	got, ok := c.Get(req)
	if !ok {
		t.Fatal("expected cache hit after Set")
	}
	if len(got) != 3 {
		t.Errorf("got %d flights, want 3", len(got))
	}
}

func TestCache_Expiry(t *testing.T) {
	c := cache.New(50 * time.Millisecond)
	req := baseReq()
	c.Set(req, makeFlights(1))

	time.Sleep(100 * time.Millisecond)
	if _, ok := c.Get(req); ok {
		t.Error("expected cache miss after TTL expiry")
	}
}

func TestCache_FilterExcludedFromKey(t *testing.T) {
	c := cache.New(time.Minute)

	req1 := baseReq()
	req1.Filters = domain.Filters{
		PriceMax: 1_000_000,
	}

	req2 := baseReq()
	req2.Filters = domain.Filters{
		PriceMax: 500_000,
	}
	req2.Sort = domain.SortPriceAsc

	c.Set(req1, makeFlights(5))

	got, ok := c.Get(req2)
	if !ok {
		t.Fatal("expected cache hit — filters and sort must not affect key")
	}
	if len(got) != 5 {
		t.Errorf("got %d flights, want 5", len(got))
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	c := cache.New(time.Minute)
	req := baseReq()
	c.Set(req, makeFlights(2))

	done := make(chan struct{})
	for i := 0; i < 50; i++ {
		go func() {
			c.Get(req)
			c.Set(req, makeFlights(2))
			done <- struct{}{}
		}()
	}
	for i := 0; i < 50; i++ {
		<-done
	}
}

func TestSearchGroup_CollapsesConcurrentRequests(t *testing.T) {
	c := cache.New(time.Minute)
	sg := cache.NewSearchGroup(c)
	req := baseReq()

	var callCount int
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sg.Do(req, func() ([]domain.Flight, error) {
				mu.Lock()
				callCount++
				mu.Unlock()
				time.Sleep(10 * time.Millisecond) // simulate aggregator latency
				return makeFlights(3), nil
			})
		}()
	}
	wg.Wait()

	if callCount != 1 {
		t.Errorf("fn called %d times, want 1 (singleflight should collapse)", callCount)
	}
}
