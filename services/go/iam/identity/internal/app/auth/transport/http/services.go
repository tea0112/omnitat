package http

import (
	"context"

	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/domains"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/transport/http/dto"
)

type AuthService interface {
	SignIn(context.Context, dto.SignInRequestDTO) (*domains.Session, error)
	SignUp(context.Context, dto.SignUpRequestDTO) (*domains.Session, error)
	Refresh(context.Context, dto.RefreshRequestDTO) (*domains.TokenPair, error)
	Logout(context.Context, dto.LogoutRequestDTO) error
}
