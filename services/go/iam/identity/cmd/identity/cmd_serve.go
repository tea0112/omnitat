package main

import (
	"context"
	"fmt"
	"net/http"
	stdHttp "net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	libDatabase "github.com/tea0112/omnitat/libs/go/database"
	"github.com/tea0112/omnitat/libs/go/datetime"
	authRepositories "github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/repositories"
	authServices "github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/services"
	authHttp "github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/transport/http"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/repositories"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/config"
)

func runServer(cfg *config.Config) error {
	db, err := libDatabase.NewDatabaseConnection(&cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}

	clock := &datetime.RealClock{}

	userRepo := repositories.NewUserRepository(db)

	// auth feature
	refreshTokenRepo := authRepositories.NewRefreshTokenRepository(redisClient)
	authService := authServices.NewAuthService(userRepo, refreshTokenRepo, clock, authServices.TokenConfig{
		JWTIssuer:       cfg.Auth.JWTIssuer,
		JWTAccessSecret: cfg.Auth.JWTAccessSecret,
		AccessTokenTTL:  cfg.Auth.AccessTokenTTL,
		RefreshTokenTTL: cfg.Auth.RefreshTokenTTL,
	})
	authHandler := authHttp.NewAuthHandler(*authService)

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	r.Get("/", func(w stdHttp.ResponseWriter, r *stdHttp.Request) {
		w.Write([]byte("identity service"))
	})

	r.Get("/healthcheck", func(w stdHttp.ResponseWriter, r *stdHttp.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("identity service OK!"))
	})

	r.Route("/api/v1", func(r chi.Router) {
		authHandler.RegisterV1(r)
	})

	server := &stdHttp.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return server.ListenAndServe()
}
