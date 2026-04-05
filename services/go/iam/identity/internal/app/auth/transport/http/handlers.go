package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	httpLib "github.com/tea0112/omnitat/libs/go/http"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/services"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/transport/http/dto"
	"github.com/tea0112/omnitat/services/go/iam/identity/pkg/apperrors"
	"github.com/tea0112/omnitat/services/go/iam/identity/pkg/exception"
)

var loginEmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type AuthHandler struct {
	authService AuthService
}

func NewAuthHandler(authServiceImpl services.AuthServiceImpl) *AuthHandler {
	return &AuthHandler{
		authService: &authServiceImpl,
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	setNoStoreHeaders(w)

	var loginRequestDTO dto.LoginRequestDTO
	err := json.NewDecoder(r.Body).Decode(&loginRequestDTO)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrInvalidJSON.Code, exception.ErrInvalidJSON.DefaultMessage))
		return
	}

	if strings.TrimSpace(loginRequestDTO.Email) == "" || loginRequestDTO.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrInvalidData.Code, "email and password are required"))
		return
	}

	if !loginEmailRegex.MatchString(strings.TrimSpace(loginRequestDTO.Email)) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrInvalidEmail.Code, exception.ErrInvalidEmail.DefaultMessage))
		return
	}

	loginDomain, err := h.authService.Login(r.Context(), loginRequestDTO)
	if err != nil {
		if errors.Is(err, apperrors.ErrInvalidCredentials) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrInvalidCredentials.Code, exception.ErrInvalidCredentials.DefaultMessage))
			return
		}

		if errors.Is(err, apperrors.ErrUserInactive) {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrUserInactive.Code, exception.ErrUserInactive.DefaultMessage))
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrLoginFailed.Code, exception.ErrLoginFailed.DefaultMessage))
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(loginDomain)
}

func setNoStoreHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}
