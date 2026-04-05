package domains

import (
	"github.com/tea0112/omnitat/libs/go/security"
	"github.com/tea0112/omnitat/services/go/iam/identity/pkg/apperrors"
)

type Account struct {
	PasswordHash *string `json:"password_hash,omitempty"`
}

func NewAccount(passwordHash *string) *Account {
	return &Account{
		PasswordHash: passwordHash,
	}
}

func (a *Account) VerifyPassword(password string) error {
	if a == nil || a.PasswordHash == nil || *a.PasswordHash == "" {
		return apperrors.ErrInvalidCredentials
	}

	match := security.VerifyPassword(password, *a.PasswordHash)
	if !match {
		return apperrors.ErrInvalidCredentials
	}

	return nil
}
