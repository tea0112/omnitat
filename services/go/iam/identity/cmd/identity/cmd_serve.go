package main

import (
	"fmt"
	stdHttp "net/http"
	"time"

	libDatabase "github.com/tea0112/omnitat/libs/go/database"
	"github.com/tea0112/omnitat/libs/go/datetime"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/repositories"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/services"
	userHttp "github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/transport/http"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/config"
	httpapi "github.com/tea0112/omnitat/services/go/iam/identity/internal/http"
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

	// Create API router with /api prefix
	apiRouter := httpapi.NewAPIRouter("/api")
	apiRouter.Register(userHandler)

	// Create main mux
	mux := stdHttp.NewServeMux()
	mux.Handle("/", apiRouter.Handler())

	// Wrap with global middleware
	handler := httpapi.Logger(httpapi.Recoverer(mux))

	server := &stdHttp.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return server.ListenAndServe()
}
