package http

import (
	"encoding/json"
	"net/http"
	"regexp"

	httpLib "github.com/tea0112/omnitat/libs/go/http"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/models"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/transport/http/dto"
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
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(dto.ErrorCode.INVALID_JSON, err.Error()))
		return
	}

	if req.Email == "" || req.Password == "" || req.FirstName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(dto.ErrorCode.INVALID_DATA, "email, password, and first_name are required"))
		return
	}

	if !emailRegex.MatchString(req.Email) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(dto.ErrorCode.INVALID_EMAIL, "invalid email format"))
		return
	}

	if len(req.Password) < MinPasswordLength {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(dto.ErrorCode.WEAK_PASSWORD, "password must be at least 8 characters"))
		return
	}

	user, err := h.userService.CreateUser(r.Context(), &req)
	if err != nil {
		code := dto.ErrorCode.CREATE_FAILED
		if err.Error() == "EMAIL_ALREADY_EXISTS" {
			code = dto.ErrorCode.EMAIL_ALREADY_EXISTS
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(code, err.Error()))
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(userToResponse(user))
}
