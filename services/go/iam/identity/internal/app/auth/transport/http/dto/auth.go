package dto

type SignInRequestDTO struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	UserAgent string `json:"-"`
	IPAddress string `json:"-"`
}

type SignUpRequestDTO struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	UserAgent string `json:"-"`
	IPAddress string `json:"-"`
}

type RefreshRequestDTO struct {
	RefreshToken string `json:"refresh_token"`
	UserAgent    string `json:"-"`
	IPAddress    string `json:"-"`
}

type LogoutRequestDTO struct {
	RefreshToken string `json:"refresh_token"`
}
