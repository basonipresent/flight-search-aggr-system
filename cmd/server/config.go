package main

import (
	"os"
	"time"
)

// Config holds all runtime configuration populated from environment variables.
type Config struct {
	Port                string
	PerProviderTimeout  time.Duration
	RequestTimeout      time.Duration
	CacheTTL            time.Duration
	ScoreWeightPrice    float64
	ScoreWeightDuration float64
	ScoreWeightStops    float64
}

// loadConfig reads environment variables and returns a Config with defaults applied.
func loadConfig() Config {
	return Config{
		Port:                getEnv("PORT", "8080"),
		PerProviderTimeout:  getDuration("PER_PROVIDER_TIMEOUT", 800*time.Millisecond),
		RequestTimeout:      getDuration("REQUEST_TIMEOUT", 2*time.Second),
		CacheTTL:            getDuration("CACHE_TTL", 60*time.Second),
		ScoreWeightPrice:    0.5,
		ScoreWeightDuration: 0.3,
		ScoreWeightStops:    0.2,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
