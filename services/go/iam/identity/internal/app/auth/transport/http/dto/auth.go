package dto

type SignInRequestDTO struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SignUpRequestDTO struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type RefreshRequestDTO struct {
	RefreshToken string `json:"refresh_token"`
}

type LogoutRequestDTO struct {
	RefreshToken string `json:"refresh_token"`
}
