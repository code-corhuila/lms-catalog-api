// Package config loads configuration from environment variables — the only source of
// environment-specific configuration (library-docs/05-architecture/cross-cutting.md).
package config

import (
	"crypto/rsa"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Config struct {
	Port string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// JWTPublicKey validates a real Administrator session token, issued by
	// lms-access-api and signed RS256 — this service never holds the private
	// key (rules/2-anexos/C-api-hexagonal.md, numeral 5.3.7).
	JWTPublicKey *rsa.PublicKey
	// InternalJWTSecret is a second, deliberately separate secret: only for
	// the short-lived HS256 tokens lms-circulation-api mints to call this
	// service's /books/{id}/loan-copy and /return-copy. This service never
	// mints one itself — it only validates.
	InternalJWTSecret string
	JWTExpiry         time.Duration

	LogLevel   string
	CORSOrigin string

	// FeatureCatalogSearchEnabled gates GET /books (week 9, config
	// hardening — "at least one feature flag guarding a new capability").
	// Default true: the capability ships on; the flag exists so it can be
	// turned off per-environment without a deploy if search needs to come
	// down (e.g. a bad query pattern hammering the DB) without taking the
	// rest of the service with it.
	FeatureCatalogSearchEnabled bool
}

func Load() (*Config, error) {
	jwtExpiry, err := time.ParseDuration(getEnv("JWT_EXPIRY", "1h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY: %w", err)
	}

	publicKeyPEM := getEnv("JWT_PUBLIC_KEY", "")
	if publicKeyPEM == "" {
		return nil, fmt.Errorf("JWT_PUBLIC_KEY must be set")
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(strings.ReplaceAll(publicKeyPEM, `\n`, "\n")))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_PUBLIC_KEY: %w", err)
	}

	cfg := &Config{
		Port: getEnv("PORT", "8080"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "lms_user"),
		DBPassword: getEnv("DB_PASSWORD", "lms_password"),
		DBName:     getEnv("DB_NAME", "lms_db"),

		JWTPublicKey:      publicKey,
		InternalJWTSecret: getEnv("INTERNAL_JWT_SECRET", ""),
		JWTExpiry:         jwtExpiry,

		LogLevel:   getEnv("LOG_LEVEL", "info"),
		CORSOrigin: getEnv("CORS_ORIGIN", "*"),

		FeatureCatalogSearchEnabled: getBoolEnv("FEATURE_CATALOG_SEARCH_ENABLED", true),
	}

	if cfg.InternalJWTSecret == "" {
		return nil, fmt.Errorf("INTERNAL_JWT_SECRET must be set")
	}

	return cfg, nil
}

// DSN builds the PostgreSQL connection string (pgx).
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return parsed
}
