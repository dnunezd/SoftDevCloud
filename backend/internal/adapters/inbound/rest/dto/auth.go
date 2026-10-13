package dto

import (
	"time"

	"github.com/dnunezd/SoftDevCloud/backend/internal/app/auth"
)

// LoginRequest es el cuerpo de POST /api/v1/auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse entrega el token de acceso y los datos del usuario autenticado.
type LoginResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresAt   time.Time    `json:"expires_at"`
	User        UserResponse `json:"user"`
}

// NewLoginResponse convierte el resultado del caso de uso.
func NewLoginResponse(r auth.LoginResult) LoginResponse {
	return LoginResponse{
		AccessToken: r.Token,
		TokenType:   "Bearer",
		ExpiresAt:   r.ExpiresAt,
		User:        NewUserResponse(r.User),
	}
}
