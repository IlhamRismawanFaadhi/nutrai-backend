package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port           string
	GinMode        string
	DSN            string
	AutoMigrate    bool
	Location       *time.Location
	AllowedOrigins []string

	N8NScanURL    string
	N8NChatURL    string
	N8NSecret     string
	N8NTimeout    time.Duration
	N8NMaxRetries int

	MaxUploadBytes int64

	FoodImageBaseURL string // mis. https://app.up.railway.app/static/foods atau URL CDN/R2
}

func Load() (*Config, error) {
	loc, err := time.LoadLocation(env("APP_TIMEZONE", "Asia/Jakarta"))
	if err != nil {
		return nil, fmt.Errorf("invalid APP_TIMEZONE: %w", err)
	}
	cfg := &Config{
		Port:           env("PORT", "8080"),
		GinMode:        env("GIN_MODE", "release"),
		AutoMigrate:    env("AUTO_MIGRATE", "true") == "true",
		Location:       loc,
		N8NScanURL:     os.Getenv("N8N_SCAN_WEBHOOK_URL"),
		N8NChatURL:     os.Getenv("N8N_CHAT_WEBHOOK_URL"),
		N8NSecret:      os.Getenv("N8N_WEBHOOK_SECRET"),
		N8NTimeout:     time.Duration(envInt("N8N_TIMEOUT_SECONDS", 45)) * time.Second,
		N8NMaxRetries:  envInt("N8N_MAX_RETRIES", 1),
		MaxUploadBytes: int64(envInt("MAX_UPLOAD_MB", 5)) << 20,

		FoodImageBaseURL: strings.TrimRight(os.Getenv("FOOD_IMAGE_BASE_URL"), "/"),
	}
	for _, o := range strings.Split(env("CORS_ALLOWED_ORIGINS", "*"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			cfg.AllowedOrigins = append(cfg.AllowedOrigins, o)
		}
	}
	cfg.DSN = buildDSN()
	if cfg.DSN == "" {
		return nil, errors.New("database not configured: set DB_DSN or MYSQL_URL")
	}
	return cfg, nil
}

const dsnParams = "charset=utf8mb4&parseTime=True&loc=UTC"

func buildDSN() string {
	if dsn := os.Getenv("DB_DSN"); dsn != "" {
		return dsn
	}
	// Railway MySQL plugin: mysql://user:pass@host:port/db
	if raw := os.Getenv("MYSQL_URL"); raw != "" {
		if u, err := url.Parse(raw); err == nil && u.User != nil {
			pass, _ := u.User.Password()
			return fmt.Sprintf("%s:%s@tcp(%s)/%s?%s", u.User.Username(), pass, u.Host, strings.TrimPrefix(u.Path, "/"), dsnParams)
		}
	}
	return ""
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil {
		return v
	}
	return def
}
