package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	httpLib "github.com/tea0112/omnitat/libs/go/http"
	"github.com/tea0112/omnitat/services/go/iam/identity/pkg/exception"
)

type authContextKey string

const userIDContextKey authContextKey = "auth.user_id"

func RequireBearerAuth(jwtSecret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenValue := bearerToken(r)
			if tokenValue == "" {
				slog.Warn("access token rejected", "event", "auth.access.rejected", "reason", "missing_bearer_token", "path", r.URL.Path, "method", r.Method, "remote_addr", r.RemoteAddr)
				writeUnauthorized(w)
				return
			}

			claims := &jwt.RegisteredClaims{}
			token, err := jwt.ParseWithClaims(tokenValue, claims, func(token *jwt.Token) (interface{}, error) {
				if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
					return nil, jwt.ErrTokenSignatureInvalid
				}
				return jwtSecret, nil
			})
			if err != nil || token == nil || !token.Valid {
				slog.Warn("access token rejected", "event", "auth.access.rejected", "reason", "invalid_token", "path", r.URL.Path, "method", r.Method, "remote_addr", r.RemoteAddr)
				writeUnauthorized(w)
				return
			}

			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				slog.Warn("access token rejected", "event", "auth.access.rejected", "reason", "invalid_subject", "path", r.URL.Path, "method", r.Method, "remote_addr", r.RemoteAddr)
				writeUnauthorized(w)
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDContextKey).(uuid.UUID)
	return userID, ok
}

func bearerToken(r *http.Request) string {
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if authorization == "" {
		return ""
	}

	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

func writeUnauthorized(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrInvalidAccessToken.Code, exception.ErrInvalidAccessToken.DefaultMessage))
}
