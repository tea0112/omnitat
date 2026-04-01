# Router Architecture Plan (Pure Go stdlib)

## Current State

```
cmd_serve.go
└── httpTransport.RegisterRoutes(mux, userHandler)  // single handler

routes.go
└── RegisterRoutes(mux *http.ServeMux, handler *UserHandler)
```

**Problem:** Adding more handlers requires modifying `RegisterRoutes` signature repeatedly.

---

## Target Architecture

```
cmd_serve.go
├── Create all handlers & services
├── Setup middleware chains per group
└── Register routers to main mux

libs/go/http/router.go
├── Router interface
├── Middleware type
└── Chain() function

Domain routers (each implements Router interface)
├── users/transport/http/routes.go
├── auth/transport/http/routes.go
└── etc.
```

---

## Step 1: Create `libs/go/http/router.go`

```go
package http

import "net/http"

// Router is implemented by domain transports to provide their routes
type Router interface {
    Routes() map[string]http.Handler
}

// Middleware wraps an http.Handler with additional behavior
type Middleware func(http.Handler) http.Handler

// Chain applies middlewares to a handler in order (left to right)
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// RegisterRoutes registers all routes from a Router to a ServeMux
// with the specified middleware applied to each route
func RegisterRoutes(mux *http.ServeMux, r Router, middlewares ...Middleware) {
	for pattern, handler := range r.Routes() {
		mux.Handle(pattern, Chain(handler, middlewares...))
	}
}
```

---

## Step 2: Refactor `users/transport/http/routes.go`

```go
package http

import (
	"net/http"
)

func (h *UserHandler) Routes() map[string]http.Handler {
	return map[string]http.Handler{
		"POST /users":       http.HandlerFunc(h.CreateUser),
		"GET /users/{id}":   http.HandlerFunc(h.GetUser),
		"PUT /users/{id}":   http.HandlerFunc(h.UpdateUser),
		"DELETE /users/{id}": http.HandlerFunc(h.DeleteUser),
		"GET /users":        http.HandlerFunc(h.ListUsers),
	}
}
```

**Note:** `UserHandler` must implement `Routes()` method.

---

## Step 3: Create Auth Domain (example)

```
internal/app/auth/
├── services/
│   ├── service.go
│   └── auth.go
├── repositories/
│   └── auth.go
└── transport/http/
    ├── handlers.go
    └── routes.go
```

**`auth/transport/http/routes.go`:**
```go
package http

import "net/http"

func (h *AuthHandler) Routes() map[string]http.Handler {
	return map[string]http.Handler{
		"POST /auth/login":  http.HandlerFunc(h.Login),
		"POST /auth/logout": http.HandlerFunc(h.Logout),
		"POST /auth/refresh": http.HandlerFunc(h.Refresh),
	}
}
```

---

## Step 4: Create Middleware Package

**`internal/http/middleware/middleware.go`:**
```go
package middleware

import (
	"log"
	"net/http"
	"time"
)

// Logger logs request details
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// Recoverer panics recovery
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Auth validates JWT token (placeholder - implement your auth logic)
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// Validate token logic here
		next.ServeHTTP(w, r)
	})
}

// RateLimiter implements simple rate limiting
func RateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Implement rate limiting logic
		next.ServeHTTP(w, r)
	})
}
```

---

## Step 5: Refactor `cmd_serve.go`

```go
package main

import (
	"fmt"
	stdHttp "net/http"
	"time"

	libDatabase "github.com/tea0112/omnitat/libs/go/database"
	"github.com/tea0112/omnitat/libs/go/datetime"
	"github.com/tea0112/omnitat/libs/go/http"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/repositories"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/services"
	httpTransport "github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/transport/http"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/config"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/http/middleware"
)

func runServer(cfg *config.Config) error {
	// 1. Database setup (unchanged)
	db, err := libDatabase.NewDatabaseConnection(&cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// 2. Create repositories (unchanged)
	userRepo := repositories.NewUserRepository(db)

	// 3. Create services
	clock := &datetime.RealClock{}
	userService := services.NewUserService(userRepo, clock)

	// 4. Create handlers
	userHandler := httpTransport.NewUserHandler(userService)

	// 5. Create main mux
	mux := stdHttp.NewServeMux()

	// 6. Register routes with appropriate middleware
	// Public routes (no auth required)
	http.RegisterRoutes(mux, userHandler)  // no middleware for public endpoints

	// Protected routes (auth required)
	http.RegisterRoutes(mux, userHandler, middleware.Auth)

	// 7. Create server with global middleware
	// Wrap mux with global middleware (logger, recoverer)
	handler := middleware.Logger(middleware.Recoverer(mux))

	server := &stdHttp.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return server.ListenAndServe()
}
```

---

## Step 6: Future Domain Addition Example

When adding a new domain (e.g., `products`):

1. Create `internal/app/products/` structure
2. Implement `Routes()` method on handler
3. In `cmd_serve.go`:

```go
// Create product handler
productHandler := productTransport.NewProductHandler(productService)

// Register with different middleware
http.RegisterRoutes(mux, productHandler, middleware.Auth, middleware.RateLimiter)
```

---

## File Changes Summary

| File | Action | Purpose |
|------|--------|---------|
| `libs/go/http/router.go` | **CREATE** | Router interface, Middleware type, Chain(), RegisterRoutes() |
| `services/.../users/transport/http/routes.go` | **MODIFY** | Add `Routes()` method returning route map |
| `services/.../auth/transport/http/routes.go` | **CREATE** | New auth domain routes |
| `internal/http/middleware/middleware.go` | **CREATE** | Logger, Recoverer, Auth, RateLimiter |
| `cmd/identity/cmd_serve.go` | **MODIFY** | Use new router registration pattern |

---

## Benefits

1. **Scalable** - Add new domains without modifying existing code
2. **Flexible middleware** - Different middleware per route group
3. **Pure stdlib** - No external dependencies
4. **Testable** - Each router can be tested in isolation
