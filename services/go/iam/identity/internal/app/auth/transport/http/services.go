package http

import (
	"context"

	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/domains"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/transport/http/dto"
)

type AuthService interface {
	Login(context.Context, dto.LoginRequestDTO) (*domains.Login, error)
	Refresh() error
	Logout() error
}
