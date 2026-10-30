package dto

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int64        `json:"expires_in"`
	User        UserResponse `json:"user"`
}

type UserResponse struct {
	ID       string  `json:"id"`
	Role     string  `json:"role"`
	Name     string  `json:"name"`
	Email    string  `json:"email"`
	Phone    *string `json:"phone,omitempty"`
	IsActive bool    `json:"is_active"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type RegisterCustomerRequest struct {
	Name     string `json:"name" binding:"required,min=3"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Phone    string `json:"phone"`
}

// GoogleLoginRequest adalah body untuk login/daftar via Google.
// IDToken didapatkan dari client setelah Google Sign-In (SDK web/mobile).
type GoogleLoginRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}
