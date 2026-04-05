package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	httpLib "github.com/tea0112/omnitat/libs/go/http"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/models"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/transport/http/dto"
	"github.com/tea0112/omnitat/services/go/iam/identity/pkg/apperrors"
	"github.com/tea0112/omnitat/services/go/iam/identity/pkg/exception"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

const MinPasswordLength = 8

type UserHandler struct {
	userService UserService
}

func NewUserHandler(userService UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func userToResponse(u *models.User) *dto.UserResponseDTO {
	return &dto.UserResponseDTO{
		Id:        u.Id,
		Email:     derefString(u.Email),
		FirstName: derefString(u.FirstName),
		LastName:  derefString(u.LastName),
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateUserDTO

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrInvalidJSON.Code, exception.ErrInvalidJSON.DefaultMessage))
		return
	}

	if req.Email == "" || req.Password == "" || req.FirstName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrInvalidData.Code, "email, password, and first_name are required"))
		return
	}

	if !emailRegex.MatchString(req.Email) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrInvalidEmail.Code, exception.ErrInvalidEmail.DefaultMessage))
		return
	}

	if len(req.Password) < MinPasswordLength {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrWeakPassword.Code, exception.ErrWeakPassword.DefaultMessage))
		return
	}

	user, err := h.userService.CreateUser(r.Context(), &req)
	if err != nil {
		if errors.Is(err, apperrors.ErrEmailAlreadyExists) {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrEmailAlreadyExists.Code, exception.ErrEmailAlreadyExists.DefaultMessage))
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrCreateFailed.Code, exception.ErrCreateFailed.DefaultMessage))
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(userToResponse(user))
}
