package dto

import (
	"time"

	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

// SessionResponse describe una sesión activa (un dispositivo).
type SessionResponse struct {
	ID        string    `json:"id"`
	UserAgent string    `json:"user_agent"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SessionListResponse es la lista de sesiones activas de un usuario.
type SessionListResponse struct {
	Items []SessionResponse `json:"items"`
}

// NewSessionListResponse convierte las sesiones de dominio.
func NewSessionListResponse(sessions []domain.Session) SessionListResponse {
	items := make([]SessionResponse, len(sessions))
	for i, s := range sessions {
		items[i] = SessionResponse{ID: s.ID, UserAgent: s.UserAgent, IP: s.IP, CreatedAt: s.CreatedAt, ExpiresAt: s.ExpiresAt}
	}
	return SessionListResponse{Items: items}
}

// RevokeSessionsResponse informa cuántas sesiones se cerraron.
type RevokeSessionsResponse struct {
	Revoked int `json:"revoked"`
}
