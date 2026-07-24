package cache

import (
	"crypto/sha256"
	"flight-search-aggr-system/internal/domain"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// entry hold a cached value and its expiry.
type entry struct {
	flights   []domain.Flight
	expiresAt time.Time
}

// Cache is an in-memory TTL cache keyed on search criteria
type Cache struct {
	mu      sync.Mutex
	entries map[string]entry
	ttl     time.Duration
}

// SearchGroup wraps Cache with singlflight to collapse concurrent identical cold-cache requests.
type SearchGroup struct {
	cache *Cache
	group singleflight.Group
}

// New creates a Cache with the given TTL.
func New(ttl time.Duration) *Cache {
	c := &Cache{
		entries: make(map[string]entry),
		ttl:     ttl,
	}
	go c.janitor()
	return c
}

// NewSearchGroup creates a SearchGroup backed by the given Cache.
func NewSearchGroup(c *Cache) *SearchGroup {
	return &SearchGroup{
		cache: c,
	}
}

// Get returns cache flights and true if a valid (non-expired) entry exists.
func (c *Cache) Get(req domain.SearchRequest) ([]domain.Flight, bool) {
	key := cacheKey(req)
	c.mu.Lock()
	e, ok := c.entries[key]
	c.mu.Unlock()
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.flights, true
}

// Set stores flights for the given request until the TTL expires.
func (c *Cache) Set(req domain.SearchRequest, flights []domain.Flight) {
	key := cacheKey(req)
	c.mu.Lock()
	c.entries[key] = entry{
		flights:   flights,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

// Do returns cached flights if available. On a miss, fn is called exactly once
// even if many goroutines call Do concurrently with the same request.
// The result is cached before returning.
func (sg *SearchGroup) Do(req domain.SearchRequest, fn func() ([]domain.Flight, error)) ([]domain.Flight, bool, error) {
	if flights, ok := sg.cache.Get(req); ok {
		return flights, true, nil
	}
	key := cacheKey(req)
	v, err, _ := sg.group.Do(key, func() (interface{}, error) {
		flights, err := fn()
		if err != nil {
			return nil, err
		}
		sg.cache.Set(req, flights)
		return flights, nil
	})
	if err != nil {
		return nil, false, err
	}
	return v.([]domain.Flight), false, nil
}

// janitor deletes expired entries every TTL interval.
func (c *Cache) janitor() {
	for {
		time.Sleep(c.ttl)
		now := time.Now()
		c.mu.Lock()
		for key, e := range c.entries {
			if now.After(e.expiresAt) {
				delete(c.entries, key)
			}
		}
		c.mu.Unlock()
	}
}

// cacheKey hashes the search criteria that determine which flights to fetch.
func cacheKey(req domain.SearchRequest) string {
	date := ""
	if !req.DepartureDate.IsZero() {
		date = req.DepartureDate.Format("2006-01-02")
	}
	raw := fmt.Sprintf("%s|%s|%s|%d|%s", req.Origin, req.Destination, date, req.Passengers, req.CabinClass)
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum)
}
