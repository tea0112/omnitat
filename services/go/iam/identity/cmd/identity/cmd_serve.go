package main

import (
	"fmt"
	stdHttp "net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	libDatabase "github.com/tea0112/omnitat/libs/go/database"
	"github.com/tea0112/omnitat/libs/go/datetime"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/repositories"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/services"
	userHttp "github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/transport/http"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/config"
)

func runServer(cfg *config.Config) error {
	db, err := libDatabase.NewDatabaseConnection(&cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	clock := &datetime.RealClock{}

	userRepo := repositories.NewUserRepository(db)
	userService := services.NewUserService(userRepo, clock)
	userHandler := userHttp.NewUserHandler(userService)

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	r.Get("/", func(w stdHttp.ResponseWriter, r *stdHttp.Request) {
		w.Write([]byte("identity service"))
	})

	r.Route("/api/v1", func(r chi.Router) {
		userHandler.RegisterV1(r)
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
