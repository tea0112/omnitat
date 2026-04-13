package exception

type HTTPErrorDef struct {
	Code           string
	DefaultMessage string
}

var ErrInvalidJSON = HTTPErrorDef{
	Code:           "INVALID_JSON",
	DefaultMessage: "request body must be valid JSON",
}

var ErrInvalidData = HTTPErrorDef{
	Code:           "INVALID_DATA",
	DefaultMessage: "request data is invalid",
}

var ErrInvalidEmail = HTTPErrorDef{
	Code:           "INVALID_EMAIL",
	DefaultMessage: "email format is invalid",
}

var ErrWeakPassword = HTTPErrorDef{
	Code:           "WEAK_PASSWORD",
	DefaultMessage: "password must be at least 8 characters",
}

var ErrEmailAlreadyExists = HTTPErrorDef{
	Code:           "EMAIL_ALREADY_EXISTS",
	DefaultMessage: "email already exists",
}

var ErrCreateFailed = HTTPErrorDef{
	Code:           "CREATE_FAILED",
	DefaultMessage: "failed to process request",
}

var ErrLoginFailed = HTTPErrorDef{
	Code:           "LOGIN_FAILED",
	DefaultMessage: "failed to process request",
}

var ErrSignupFailed = HTTPErrorDef{
	Code:           "SIGNUP_FAILED",
	DefaultMessage: "failed to process request",
}

var ErrUserInactive = HTTPErrorDef{
	Code:           "USER_INACTIVE",
	DefaultMessage: "user account is inactive",
}

var ErrInvalidCredentials = HTTPErrorDef{
	Code:           "INVALID_CREDENTIALS",
	DefaultMessage: "invalid credentials",
}

var ErrInvalidRefreshToken = HTTPErrorDef{
	Code:           "INVALID_REFRESH_TOKEN",
	DefaultMessage: "invalid refresh token",
}

var ErrRefreshFailed = HTTPErrorDef{
	Code:           "REFRESH_FAILED",
	DefaultMessage: "failed to process request",
}

var ErrLogoutFailed = HTTPErrorDef{
	Code:           "LOGOUT_FAILED",
	DefaultMessage: "failed to process request",
}
