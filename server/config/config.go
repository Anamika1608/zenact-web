package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	OpenRouterAPIKey string
	OpenRouterModel  string
	BrowserHeadless  bool
	BrowserWidth     int
	BrowserHeight    int
	MaxIterations    int
	ServerPort       string
}

func Load() (*Config, error) {
	godotenv.Load()

	cfg := &Config{
		OpenRouterAPIKey: os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterModel:  getEnvOrDefault("OPENROUTER_MODEL", "google/gemini-2.0-flash-001"),
		BrowserHeadless:  getEnvOrDefault("BROWSER_HEADLESS", "false") == "true",
		BrowserWidth:     getEnvInt("BROWSER_WIDTH", 1280),
		BrowserHeight:    getEnvInt("BROWSER_HEIGHT", 900),
		MaxIterations:    getEnvInt("MAX_ITERATIONS", 30),
		ServerPort:       getEnvOrDefault("SERVER_PORT", "8080"),
	}

	if cfg.OpenRouterAPIKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY is required")
	}
	return cfg, nil
}

func getEnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return n
}
