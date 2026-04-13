package apperrors

import "errors"

var ErrEmailAlreadyExists = errors.New("email already exists")

var ErrInvalidCredentials = errors.New("invalid credentials")

var ErrUserInactive = errors.New("user inactive")

var ErrInvalidRefreshToken = errors.New("invalid refresh token")

var ErrMapNilUserDomain = errors.New("map user domain: nil user")
