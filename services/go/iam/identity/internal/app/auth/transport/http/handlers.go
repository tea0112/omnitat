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

const MinPasswordLength = 8

type AuthHandler struct {
	authService AuthService
}

func NewAuthHandler(authServiceImpl services.AuthServiceImpl) *AuthHandler {
	return &AuthHandler{
		authService: &authServiceImpl,
	}
}

func (h *AuthHandler) SignIn(w http.ResponseWriter, r *http.Request) {
	setNoStoreHeaders(w)

	var signInRequestDTO dto.SignInRequestDTO
	err := json.NewDecoder(r.Body).Decode(&signInRequestDTO)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrInvalidJSON.Code, exception.ErrInvalidJSON.DefaultMessage))
		return
	}

	if strings.TrimSpace(signInRequestDTO.Email) == "" || signInRequestDTO.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrInvalidData.Code, "email and password are required"))
		return
	}

	if !loginEmailRegex.MatchString(strings.TrimSpace(signInRequestDTO.Email)) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrInvalidEmail.Code, exception.ErrInvalidEmail.DefaultMessage))
		return
	}

	session, err := h.authService.SignIn(r.Context(), signInRequestDTO)
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
	json.NewEncoder(w).Encode(session)
}

func (h *AuthHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	setNoStoreHeaders(w)

	var signUpRequestDTO dto.SignUpRequestDTO
	err := json.NewDecoder(r.Body).Decode(&signUpRequestDTO)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrInvalidJSON.Code, exception.ErrInvalidJSON.DefaultMessage))
		return
	}

	if strings.TrimSpace(signUpRequestDTO.Email) == "" || signUpRequestDTO.Password == "" || strings.TrimSpace(signUpRequestDTO.FirstName) == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrInvalidData.Code, "email, password, and first_name are required"))
		return
	}

	if !loginEmailRegex.MatchString(strings.TrimSpace(signUpRequestDTO.Email)) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrInvalidEmail.Code, exception.ErrInvalidEmail.DefaultMessage))
		return
	}

	if len(signUpRequestDTO.Password) < MinPasswordLength {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrWeakPassword.Code, exception.ErrWeakPassword.DefaultMessage))
		return
	}

	session, err := h.authService.SignUp(r.Context(), signUpRequestDTO)
	if err != nil {
		if errors.Is(err, apperrors.ErrEmailAlreadyExists) {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrEmailAlreadyExists.Code, exception.ErrEmailAlreadyExists.DefaultMessage))
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrSignupFailed.Code, exception.ErrSignupFailed.DefaultMessage))
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(session)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	setNoStoreHeaders(w)

	var refreshRequestDTO dto.RefreshRequestDTO
	err := json.NewDecoder(r.Body).Decode(&refreshRequestDTO)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrInvalidJSON.Code, exception.ErrInvalidJSON.DefaultMessage))
		return
	}

	if strings.TrimSpace(refreshRequestDTO.RefreshToken) == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrInvalidData.Code, "refresh_token is required"))
		return
	}

	tokenPair, err := h.authService.Refresh(r.Context(), refreshRequestDTO)
	if err != nil {
		if errors.Is(err, apperrors.ErrInvalidRefreshToken) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrInvalidRefreshToken.Code, exception.ErrInvalidRefreshToken.DefaultMessage))
			return
		}

		if errors.Is(err, apperrors.ErrUserInactive) {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrUserInactive.Code, exception.ErrUserInactive.DefaultMessage))
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrRefreshFailed.Code, exception.ErrRefreshFailed.DefaultMessage))
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tokenPair)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	setNoStoreHeaders(w)

	var logoutRequestDTO dto.LogoutRequestDTO
	err := json.NewDecoder(r.Body).Decode(&logoutRequestDTO)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrInvalidJSON.Code, exception.ErrInvalidJSON.DefaultMessage))
		return
	}

	if strings.TrimSpace(logoutRequestDTO.RefreshToken) == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrInvalidData.Code, "refresh_token is required"))
		return
	}

	err = h.authService.Logout(r.Context(), logoutRequestDTO)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrLogoutFailed.Code, exception.ErrLogoutFailed.DefaultMessage))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func setNoStoreHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}
