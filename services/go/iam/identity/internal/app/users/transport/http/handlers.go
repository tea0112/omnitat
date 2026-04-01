package http

import (
	"encoding/json"
	"net/http"

	libHttp "github.com/tea0112/omnitat/libs/go/http"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/transport/http/dto"
)

type UserHandler struct {
	userService UserService
}

func NewUserHandler(userService UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateUserDTO

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).
			Encode(libHttp.ErrorResponseCode("INVALID_JSON", err.Error()))
		return
	}

	// TODO: validate request
	if req.Email == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).
			Encode(libHttp.ErrorResponseCode("INVALID_DATA", "invalid data"))
		return
	}

	user, err := h.userService.CreateUser(r.Context(), &req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).
			Encode(libHttp.ErrorResponseCode("CREATE_FAILED", err.Error()))
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}
