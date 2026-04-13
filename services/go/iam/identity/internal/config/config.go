package config

import (
	"time"

	"github.com/tea0112/omnitat/libs/go/config"
	libDatabase "github.com/tea0112/omnitat/libs/go/database"
	libHttp "github.com/tea0112/omnitat/libs/go/http"
)

type Config struct {
	HTTP     libHttp.Config
	Database libDatabase.DatabaseConfig
	Redis    RedisConfig
	Auth     AuthConfig
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type AuthConfig struct {
	JWTIssuer       string
	JWTAccessSecret string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

func Load() (*Config, error) {
	cfg := Config{}

	// HTTP Server config
	cfg.HTTP.Port = config.GetEnv("HTTP_PORT", 8881)

	// Database config
	cfg.Database.Host = config.GetEnv("DB_HOST", "localhost")
	cfg.Database.Port = config.GetEnv("DB_PORT", 5432)
	cfg.Database.User = config.GetEnv("DB_USER", "postgres")
	cfg.Database.Password = config.GetEnv("DB_PASSWORD", "secret")
	cfg.Database.DatabaseName = config.GetEnv("DB_DATABASE_NAME", "identity")
	cfg.Database.SslMode = config.GetEnv("DB_SSL_MODE", "disable")
	cfg.Database.MaxOpenConns = config.GetEnv("DB_MAX_OPEN_CONNS", 25)
	cfg.Database.MaxIdleConns = config.GetEnv("DB_MAX_IDLE_CONNS", 25)

	dbConnMaxLifetimeStr := config.GetEnv("DB_CONN_MAX_LIFETIME", "5m")
	dbConnMaxLifetimeDuration, err := time.ParseDuration(dbConnMaxLifetimeStr)
	if err != nil {
		return nil, err
	}
	cfg.Database.ConnMaxLifetime = dbConnMaxLifetimeDuration

	dbConnMaxIdleTimeStr := config.GetEnv("DB_CONN_MAX_IDLE_TIME", "3m")
	dbConnMaxIdleTimeDuration, err := time.ParseDuration(dbConnMaxIdleTimeStr)
	if err != nil {
		return nil, err
	}
	cfg.Database.ConnMaxIdleTime = dbConnMaxIdleTimeDuration

	// Auth config
	cfg.Redis.Addr = config.GetEnv("REDIS_ADDR", "localhost:6379")
	cfg.Redis.Password = config.GetEnv("REDIS_PASSWORD", "")
	cfg.Redis.DB = config.GetEnv("REDIS_DB", 0)

	cfg.Auth.JWTIssuer = config.GetEnv("JWT_ISSUER", "identity-service")
	cfg.Auth.JWTAccessSecret = config.GetEnv("JWT_ACCESS_SECRET", "dev-access-secret")

	accessTokenTTLStr := config.GetEnv("JWT_ACCESS_TTL", "15m")
	accessTokenTTL, err := time.ParseDuration(accessTokenTTLStr)
	if err != nil {
		return nil, err
	}
	cfg.Auth.AccessTokenTTL = accessTokenTTL

	refreshTokenTTLStr := config.GetEnv("REFRESH_TOKEN_TTL", "720h")
	refreshTokenTTL, err := time.ParseDuration(refreshTokenTTLStr)
	if err != nil {
		return nil, err
	}
	cfg.Auth.RefreshTokenTTL = refreshTokenTTL

	return &cfg, nil
}
