package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Address                  string
	SchemaDirectory          string
	CancellationWindow       time.Duration
	CancellationWindowNFCe   time.Duration
	MaxDistributionDocuments int
}

func Load() Config {
	return Config{
		Address:                  text("FAKE_SEFAZ_ADDRESS", ":8080"),
		SchemaDirectory:          text("FAKE_SEFAZ_SCHEMA_DIR", ""),
		CancellationWindow:       duration("FAKE_SEFAZ_CANCELLATION_WINDOW", 24*time.Hour),
		CancellationWindowNFCe:   duration("FAKE_SEFAZ_CANCELLATION_WINDOW_NFCE", 30*time.Minute),
		MaxDistributionDocuments: number("FAKE_SEFAZ_MAX_DISTRIBUTION_DOCUMENTS", 50),
	}
}

func text(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func number(name string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil {
		return fallback
	}
	return value
}

func duration(name string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(os.Getenv(name))
	if err != nil {
		return fallback
	}
	return value
}
