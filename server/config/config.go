package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// MinJWTSecretLen is the minimum length (in bytes) accepted for JWT_SECRET.
// HS256 is only as strong as its key: 32 bytes matches the hash output size.
const MinJWTSecretLen = 32

type Config struct {
	Port            int
	DatabaseURL     string
	JWTSecret       string
	AccessTokenTTL  time.Duration // lifetime of the JWT access token
	RefreshTokenTTL time.Duration // lifetime of a refresh token (rotated on use)
	LoginRateLimit  int           // max calls per minute and per IP on /auth/*
	UploadDir       string
	MaxUploadSize   int64 // bytes
}

// Load reads the configuration from the environment. DATABASE_URL and
// JWT_SECRET have no default: a misconfigured deployment must fail to start
// rather than run with guessable credentials.
func Load() (*Config, error) {
	port := 8080
	if v := os.Getenv("PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid PORT: %w", err)
		}
		port = p
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < MinJWTSecretLen {
		return nil, fmt.Errorf("JWT_SECRET is required and must be at least %d characters (got %d)", MinJWTSecretLen, len(jwtSecret))
	}

	accessTTL, err := durationEnv("ACCESS_TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return nil, err
	}
	refreshTTL, err := durationEnv("REFRESH_TOKEN_TTL", 30*24*time.Hour)
	if err != nil {
		return nil, err
	}

	loginRateLimit := 10
	if v := os.Getenv("LOGIN_RATE_LIMIT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return nil, fmt.Errorf("invalid LOGIN_RATE_LIMIT: %q (positive integer expected)", v)
		}
		loginRateLimit = n
	}

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "uploads"
	}

	maxUploadSize := int64(10 << 20) // 10 MB default
	if v := os.Getenv("MAX_UPLOAD_SIZE"); v != "" {
		s, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid MAX_UPLOAD_SIZE: %w", err)
		}
		maxUploadSize = s
	}

	return &Config{
		Port:            port,
		DatabaseURL:     dbURL,
		JWTSecret:       jwtSecret,
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
		LoginRateLimit:  loginRateLimit,
		UploadDir:       uploadDir,
		MaxUploadSize:   maxUploadSize,
	}, nil
}

func durationEnv(name string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(name)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("invalid %s: %q (positive Go duration expected, e.g. 15m, 24h)", name, v)
	}
	return d, nil
}
