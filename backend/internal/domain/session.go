package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

// MaxUserAgentLength limita el agente de usuario que se guarda por sesión.
const MaxUserAgentLength = 255

// Session es un inicio de sesión activo, uno por dispositivo. Cerrar sesión la
// revoca sin afectar las demás sesiones del mismo usuario.
type Session struct {
	ID        string
	UserID    string
	UserAgent string
	IP        string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// NewSession crea una sesión que vence ttl después de now. El agente de usuario
// viene de una cabecera HTTP, que puede traer bytes que no son UTF-8 (HTTP
// admite Latin-1): se reemplazan por U+FFFD, porque PostgreSQL rechaza texto
// inválido, y se recorta para no guardar cabeceras arbitrariamente largas.
func NewSession(userID, userAgent, ip string, now time.Time, ttl time.Duration) Session {
	userAgent = strings.ToValidUTF8(userAgent, "�")
	if utf8.RuneCountInString(userAgent) > MaxUserAgentLength {
		userAgent = string([]rune(userAgent)[:MaxUserAgentLength])
	}
	return Session{UserID: userID, UserAgent: userAgent, IP: ip, ExpiresAt: now.Add(ttl)}
}
