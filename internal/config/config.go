package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv        string
	Port          string
	SessionSecret string
	AdminWhatsApp string
	DatabaseURL   string // belum dipakai pada tahap tampilan
}

func (c *Config) IsProduction() bool { return c.AppEnv == "production" }

func Load() (*Config, error) {
	_ = godotenv.Load() // .env opsional

	cfg := &Config{
		AppEnv:        get("APP_ENV", "development"),
		Port:          get("PORT", "8080"),
		SessionSecret: os.Getenv("SESSION_SECRET"),
		AdminWhatsApp: os.Getenv("ADMIN_WHATSAPP"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
	}

	// Mode development boleh jalan tanpa .env sama sekali.
	if cfg.SessionSecret == "" && !cfg.IsProduction() {
		cfg.SessionSecret = "dev-only-secret-jangan-dipakai-di-produksi"
	}
	if len(cfg.SessionSecret) < 32 {
		return nil, errors.New("SESSION_SECRET minimal 32 karakter")
	}
	return cfg, nil
}

func get(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
