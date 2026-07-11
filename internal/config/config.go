package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port        string
	Environment string
	LogLevel    string

	DatabaseURL string

	JWTSecret          string
	JWTAccessTTLMin    int
	JWTRefreshTTLDays  int

	MailHost     string
	MailPort     int
	MailUsername string
	MailPassword string
	MailFrom     string
}

func Load() Config {
	return Config{
		Port:        getEnv("PORT", "8083"),
		Environment: getEnv("ENVIRONMENT", "dev"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),

		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5460/backtemplate?sslmode=disable"),

		JWTSecret:         getEnv("JWT_SECRET", "change-me-to-a-long-random-string-in-real-deployments"),
		JWTAccessTTLMin:   getEnvInt("JWT_ACCESS_TTL_MINUTES", 15),
		JWTRefreshTTLDays: getEnvInt("JWT_REFRESH_TTL_DAYS", 30),

		MailHost:     getEnv("MAIL_HOST", ""),
		MailPort:     getEnvInt("MAIL_PORT", 587),
		MailUsername: getEnv("MAIL_USERNAME", ""),
		MailPassword: getEnv("MAIL_PASSWORD", ""),
		MailFrom:     getEnv("MAIL_FROM", "no-reply@example.com"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
