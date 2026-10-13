// Package auth implementa el inicio de sesión local, la validación de tokens y
// la gestión de sesiones (RF-01, DT-03).
package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dnunezd/SoftDevCloud/backend/internal/app/access"
	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
	"github.com/dnunezd/SoftDevCloud/backend/internal/ports"
)

// Service autentica usuarios y administra sus sesiones.
type Service struct {
	users      ports.UserRepository
	sessions   ports.SessionRepository
	hasher     ports.PasswordHasher
	tokens     ports.TokenService
	clock      ports.Clock
	sessionTTL time.Duration
}

// NewService crea el servicio. sessionTTL es la duración de cada sesión (JWT_TTL).
func NewService(
	users ports.UserRepository,
	sessions ports.SessionRepository,
	hasher ports.PasswordHasher,
	tokens ports.TokenService,
	clock ports.Clock,
	sessionTTL time.Duration,
) *Service {
	return &Service{users: users, sessions: sessions, hasher: hasher, tokens: tokens, clock: clock, sessionTTL: sessionTTL}
}

// Client describe el dispositivo desde el que se inicia sesión.
type Client struct {
	UserAgent string
	IP        string
}

// LoginResult es el resultado de un inicio de sesión exitoso.
type LoginResult struct {
	Token     string
	ExpiresAt time.Time
	User      domain.User
}

// Login autentica con correo y contraseña (RF-01.1) y abre una sesión nueva para
// el dispositivo. Ante un correo inexistente o una contraseña incorrecta
// devuelve el mismo error, para no revelar qué correos están registrados.
func (s *Service) Login(ctx context.Context, email, password string, client Client) (LoginResult, error) {
	fields := map[string]string{}
	if strings.TrimSpace(email) == "" {
		fields["email"] = "es obligatorio"
	}
	if password == "" {
		fields["password"] = "es obligatoria"
	}
	if len(fields) > 0 {
		return LoginResult{}, domain.NewValidationError(fields)
	}

	user, err := s.users.GetByEmail(ctx, domain.NormalizeEmail(email))
	if errors.Is(err, domain.ErrUserNotFound) {
		return LoginResult{}, domain.ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, fmt.Errorf("buscar usuario: %w", err)
	}

	ok, err := s.hasher.Compare(user.PasswordHash, password)
	if err != nil {
		return LoginResult{}, fmt.Errorf("verificar contraseña: %w", err)
	}
	if !ok {
		return LoginResult{}, domain.ErrInvalidCredentials
	}

	session, err := s.sessions.Create(ctx,
		domain.NewSession(user.ID, client.UserAgent, client.IP, s.clock.Now(), s.sessionTTL))
	if err != nil {
		return LoginResult{}, fmt.Errorf("abrir sesión: %w", err)
	}
	token, err := s.tokens.Issue(ports.TokenClaims{UserID: user.ID, SessionID: session.ID, ExpiresAt: session.ExpiresAt})
	if err != nil {
		return LoginResult{}, fmt.Errorf("emitir token: %w", err)
	}
	return LoginResult{Token: token, ExpiresAt: session.ExpiresAt, User: user}, nil
}

// Authenticate valida el token, verifica que su sesión siga activa y carga los
// roles vigentes del usuario (RF-01.2). Así, un cierre de sesión o un cambio de
// roles aplican en la siguiente petición.
//
// Un token inválido o expirado produce domain.ErrUnauthenticated; una sesión
// cerrada o vencida, domain.ErrSessionRevoked. Una falla de la base de datos se
// propaga tal cual (500), para no confundirla con un problema de credenciales.
func (s *Service) Authenticate(ctx context.Context, token string) (access.Principal, error) {
	claims, err := s.tokens.Parse(token)
	if err != nil {
		return access.Principal{}, domain.ErrUnauthenticated
	}

	roles, err := s.sessions.ActiveRoles(ctx, claims.SessionID, claims.UserID)
	if errors.Is(err, domain.ErrSessionNotFound) {
		return access.Principal{}, domain.ErrSessionRevoked
	}
	if err != nil {
		return access.Principal{}, fmt.Errorf("verificar sesión: %w", err)
	}
	return access.Principal{UserID: claims.UserID, SessionID: claims.SessionID, Roles: roles}, nil
}

// Logout cierra solo la sesión desde la que se hace la petición; las sesiones
// del mismo usuario en otros dispositivos siguen activas.
func (s *Service) Logout(ctx context.Context, principal access.Principal) error {
	if err := s.sessions.Revoke(ctx, principal.SessionID); err != nil {
		return fmt.Errorf("cerrar sesión: %w", err)
	}
	return nil
}

// ListSessions devuelve las sesiones activas de un usuario (uso del administrador).
func (s *Service) ListSessions(ctx context.Context, userID string) ([]domain.Session, error) {
	if _, err := s.users.GetByID(ctx, userID); err != nil {
		return nil, fmt.Errorf("consultar usuario: %w", err)
	}
	sessions, err := s.sessions.ListActive(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listar sesiones: %w", err)
	}
	return sessions, nil
}

// RevokeSessions cierra todas las sesiones de un usuario, por ejemplo si su token
// fue robado (uso del administrador). Devuelve cuántas sesiones cerró.
func (s *Service) RevokeSessions(ctx context.Context, userID string) (int, error) {
	if _, err := s.users.GetByID(ctx, userID); err != nil {
		return 0, fmt.Errorf("consultar usuario: %w", err)
	}
	revoked, err := s.sessions.RevokeAll(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("cerrar sesiones: %w", err)
	}
	return revoked, nil
}

// PurgeExpiredSessions borra las sesiones vencidas de todos los usuarios. Se
// ejecuta al arrancar el proceso.
func (s *Service) PurgeExpiredSessions(ctx context.Context) (int, error) {
	n, err := s.sessions.DeleteExpired(ctx)
	if err != nil {
		return 0, fmt.Errorf("borrar sesiones vencidas: %w", err)
	}
	return n, nil
}
