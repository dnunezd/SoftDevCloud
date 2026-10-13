package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dnunezd/SoftDevCloud/backend/internal/app/access"
	"github.com/dnunezd/SoftDevCloud/backend/internal/app/auth"
	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
	"github.com/dnunezd/SoftDevCloud/backend/internal/mocks"
	"github.com/dnunezd/SoftDevCloud/backend/internal/ports"
)

const (
	sessionID  = "5f0b9c1e-2d3a-4b5c-8d9e-0f1a2b3c4d5e"
	sessionTTL = 8 * time.Hour
)

var (
	now        = time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)
	storedUser = domain.User{
		ID:           "8c1f4f0e-3f5b-4d36-9a52-1d2f1f0e8a11",
		FullName:     "Ana Pérez",
		Email:        "ana@uni.edu",
		PasswordHash: "hash-guardado",
		Roles:        []domain.Role{domain.RoleGraduateAssistant, domain.RoleMonitor},
	}
	client = auth.Client{UserAgent: "Mozilla/5.0 (portátil)", IP: "10.0.0.7"}
)

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

type fixture struct {
	service  *auth.Service
	users    *mocks.UserRepository
	sessions *mocks.SessionRepository
	hasher   *mocks.PasswordHasher
	tokens   *mocks.TokenService
}

func newFixture(t *testing.T) fixture {
	f := fixture{
		users:    mocks.NewUserRepository(t),
		sessions: mocks.NewSessionRepository(t),
		hasher:   mocks.NewPasswordHasher(t),
		tokens:   mocks.NewTokenService(t),
	}
	f.service = auth.NewService(f.users, f.sessions, f.hasher, f.tokens, fixedClock{now}, sessionTTL)
	return f
}

func TestLogin_OpensASessionForTheDevice(t *testing.T) {
	f := newFixture(t)
	expiresAt := now.Add(sessionTTL)
	f.users.EXPECT().GetByEmail(mock.Anything, "ana@uni.edu").Return(storedUser, nil)
	f.hasher.EXPECT().Compare("hash-guardado", "secreta123").Return(true, nil)
	f.sessions.EXPECT().Create(mock.Anything, domain.Session{
		UserID: storedUser.ID, UserAgent: client.UserAgent, IP: client.IP, ExpiresAt: expiresAt,
	}).Return(domain.Session{ID: sessionID, UserID: storedUser.ID, ExpiresAt: expiresAt}, nil)
	f.tokens.EXPECT().Issue(ports.TokenClaims{UserID: storedUser.ID, SessionID: sessionID, ExpiresAt: expiresAt}).
		Return("token-firmado", nil)

	result, err := f.service.Login(context.Background(), "  Ana@Uni.EDU ", "secreta123", client)

	require.NoError(t, err)
	assert.Equal(t, auth.LoginResult{Token: "token-firmado", ExpiresAt: expiresAt, User: storedUser}, result)
}

func TestLogin_UnknownEmailAndWrongPasswordReturnTheSameError(t *testing.T) {
	t.Run("correo inexistente", func(t *testing.T) {
		f := newFixture(t)
		f.users.EXPECT().GetByEmail(mock.Anything, "nadie@uni.edu").Return(domain.User{}, domain.ErrUserNotFound)

		_, err := f.service.Login(context.Background(), "nadie@uni.edu", "secreta123", client)

		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("contraseña incorrecta no abre sesión", func(t *testing.T) {
		f := newFixture(t)
		f.users.EXPECT().GetByEmail(mock.Anything, "ana@uni.edu").Return(storedUser, nil)
		f.hasher.EXPECT().Compare("hash-guardado", "otra-clave").Return(false, nil)

		_, err := f.service.Login(context.Background(), "ana@uni.edu", "otra-clave", client)

		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})
}

func TestLogin_RequiresEmailAndPassword(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.Login(context.Background(), " ", "", client)

	var derr *domain.Error
	require.ErrorAs(t, err, &derr)
	assert.Equal(t, domain.KindValidation, derr.Kind)
	assert.Contains(t, derr.Fields, "email")
	assert.Contains(t, derr.Fields, "password")
}

func TestLogin_InfrastructureFailureIsNotReportedAsInvalidCredentials(t *testing.T) {
	f := newFixture(t)
	dbDown := errors.New("conexión rechazada")
	f.users.EXPECT().GetByEmail(mock.Anything, "ana@uni.edu").Return(domain.User{}, dbDown)

	_, err := f.service.Login(context.Background(), "ana@uni.edu", "secreta123", client)

	assert.ErrorIs(t, err, dbDown)
	assert.NotErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthenticate(t *testing.T) {
	ctx := context.Background()
	validClaims := ports.TokenClaims{UserID: storedUser.ID, SessionID: sessionID}

	t.Run("sesión activa: identidad con los roles vigentes en la base de datos", func(t *testing.T) {
		f := newFixture(t)
		f.tokens.EXPECT().Parse("token-firmado").Return(validClaims, nil)
		// Al iniciar sesión era monitor y asistente; desde entonces el administrador lo cambió.
		f.sessions.EXPECT().ActiveRoles(mock.Anything, sessionID, storedUser.ID).Return([]domain.Role{domain.RoleProfessor}, nil)

		principal, err := f.service.Authenticate(ctx, "token-firmado")

		require.NoError(t, err)
		assert.Equal(t, access.Principal{
			UserID: storedUser.ID, SessionID: sessionID, Roles: []domain.Role{domain.RoleProfessor},
		}, principal)
	})

	t.Run("token inválido no consulta la base de datos", func(t *testing.T) {
		f := newFixture(t)
		f.tokens.EXPECT().Parse("token-alterado").Return(ports.TokenClaims{}, errors.New("firma inválida"))

		_, err := f.service.Authenticate(ctx, "token-alterado")

		assert.ErrorIs(t, err, domain.ErrUnauthenticated)
	})

	t.Run("sesión cerrada, vencida o de un usuario eliminado", func(t *testing.T) {
		f := newFixture(t)
		f.tokens.EXPECT().Parse("token-firmado").Return(validClaims, nil)
		f.sessions.EXPECT().ActiveRoles(mock.Anything, sessionID, storedUser.ID).Return(nil, domain.ErrSessionNotFound)

		_, err := f.service.Authenticate(ctx, "token-firmado")

		assert.ErrorIs(t, err, domain.ErrSessionRevoked)
	})

	t.Run("una falla de la base de datos no se reporta como sesión inválida", func(t *testing.T) {
		f := newFixture(t)
		dbDown := errors.New("conexión rechazada")
		f.tokens.EXPECT().Parse("token-firmado").Return(validClaims, nil)
		f.sessions.EXPECT().ActiveRoles(mock.Anything, sessionID, storedUser.ID).Return(nil, dbDown)

		_, err := f.service.Authenticate(ctx, "token-firmado")

		assert.ErrorIs(t, err, dbDown)
		assert.NotErrorIs(t, err, domain.ErrSessionRevoked)
	})
}

func TestLogout_ClosesOnlyTheCurrentSession(t *testing.T) {
	f := newFixture(t)
	f.sessions.EXPECT().Revoke(mock.Anything, sessionID).Return(nil)

	err := f.service.Logout(context.Background(), access.Principal{UserID: storedUser.ID, SessionID: sessionID})

	require.NoError(t, err)
	// mockery verifica al final que no se llamó a RevokeAll.
}

func TestAdminSessionManagement(t *testing.T) {
	ctx := context.Background()

	t.Run("lista las sesiones activas del usuario", func(t *testing.T) {
		f := newFixture(t)
		active := []domain.Session{{ID: sessionID, UserID: storedUser.ID}}
		f.users.EXPECT().GetByID(mock.Anything, storedUser.ID).Return(storedUser, nil)
		f.sessions.EXPECT().ListActive(mock.Anything, storedUser.ID).Return(active, nil)

		sessions, err := f.service.ListSessions(ctx, storedUser.ID)

		require.NoError(t, err)
		assert.Equal(t, active, sessions)
	})

	t.Run("cierra todas las sesiones del usuario", func(t *testing.T) {
		f := newFixture(t)
		f.users.EXPECT().GetByID(mock.Anything, storedUser.ID).Return(storedUser, nil)
		f.sessions.EXPECT().RevokeAll(mock.Anything, storedUser.ID).Return(2, nil)

		revoked, err := f.service.RevokeSessions(ctx, storedUser.ID)

		require.NoError(t, err)
		assert.Equal(t, 2, revoked)
	})

	t.Run("usuario inexistente", func(t *testing.T) {
		f := newFixture(t)
		f.users.EXPECT().GetByID(mock.Anything, "no-existe").Return(domain.User{}, domain.ErrUserNotFound).Twice()

		_, err := f.service.ListSessions(ctx, "no-existe")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		_, err = f.service.RevokeSessions(ctx, "no-existe")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("limpia las sesiones vencidas", func(t *testing.T) {
		f := newFixture(t)
		f.sessions.EXPECT().DeleteExpired(mock.Anything).Return(3, nil)

		purged, err := f.service.PurgeExpiredSessions(ctx)

		require.NoError(t, err)
		assert.Equal(t, 3, purged)
	})
}
