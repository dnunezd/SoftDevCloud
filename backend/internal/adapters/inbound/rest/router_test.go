package rest_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/inbound/rest"
	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/inbound/rest/dto"
	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/outbound/clock"
	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/outbound/security"
	"github.com/dnunezd/SoftDevCloud/backend/internal/app/auth"
	"github.com/dnunezd/SoftDevCloud/backend/internal/app/periods"
	"github.com/dnunezd/SoftDevCloud/backend/internal/app/users"
	"github.com/dnunezd/SoftDevCloud/backend/internal/app/workspaces"
	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
	"github.com/dnunezd/SoftDevCloud/backend/internal/mocks"
	"github.com/dnunezd/SoftDevCloud/backend/internal/ports"
)

const (
	adminID     = "0b7c1c52-6a0e-4b63-8f0a-5a5f3b2d1c01"
	monitorID   = "1d2e3f40-5a6b-4c7d-8e9f-a0b1c2d3e4f5"
	betoID      = "2a3b4c5d-6e7f-4a8b-9c0d-1e2f3a4b5c6d"
	profID      = "3b4c5d6e-7f80-4b9c-8d1e-2f3a4b5c6d7e"
	otherProfID = "4c5d6e7f-8091-4cad-9e2f-3a4b5c6d7e8f"
	userAgent   = "pruebas/1.0"
)

// fixedClock fija la hora del caso de uso de espacios: 10 de febrero de 2026,
// dentro de la ventana de creación del período de las pruebas.
type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

var testNow = time.Date(2026, time.February, 10, 12, 0, 0, 0, time.UTC)

// device es una sesión abierta por un usuario en un dispositivo.
type device struct{ userID, sessionID string }

var (
	adminLaptop = device{adminID, "a0000000-0000-4000-8000-000000000001"}
	anaLaptop   = device{monitorID, "a0000000-0000-4000-8000-000000000002"}
	anaPhone    = device{monitorID, "a0000000-0000-4000-8000-000000000003"}
	betoLaptop  = device{betoID, "a0000000-0000-4000-8000-000000000004"}
	profLaptop  = device{profID, "a0000000-0000-4000-8000-000000000005"}
	otherProf   = device{otherProfID, "a0000000-0000-4000-8000-000000000006"}
)

// testAPI levanta el router real con los casos de uso reales; solo los
// repositorios están simulados. bcrypt y JWT son los adaptadores reales.
type testAPI struct {
	router     http.Handler
	repo       *mocks.UserRepository
	sessions   *mocks.SessionRepository
	periods    *mocks.PeriodRepository
	workspaces *mocks.WorkspaceRepository
	hasher     *security.BcryptHasher
	tokens     *security.JWTService
}

func newTestAPI(t *testing.T) testAPI {
	t.Helper()
	gin.SetMode(gin.TestMode)
	api := testAPI{
		repo:       mocks.NewUserRepository(t),
		sessions:   mocks.NewSessionRepository(t),
		periods:    mocks.NewPeriodRepository(t),
		workspaces: mocks.NewWorkspaceRepository(t),
		hasher:     security.NewBcryptHasher(bcrypt.MinCost),
		tokens:     security.NewJWTService("clave-de-pruebas-de-al-menos-32-caracteres", time.Now),
	}
	bogota, err := time.LoadLocation("America/Bogota")
	require.NoError(t, err)
	router, err := rest.NewRouter(rest.Dependencies{
		Auth:       auth.NewService(api.repo, api.sessions, api.hasher, api.tokens, clock.System{}, time.Hour),
		Users:      users.NewService(api.repo, api.hasher),
		Periods:    periods.NewService(api.periods),
		Workspaces: workspaces.NewService(api.workspaces, api.periods, fixedClock{testNow}, bogota),
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	require.NoError(t, err)
	api.router = router
	return api
}

// tokenFor emite el token de la sesión de un dispositivo.
func (a testAPI) tokenFor(t *testing.T, d device) string {
	t.Helper()
	token, err := a.tokens.Issue(ports.TokenClaims{UserID: d.userID, SessionID: d.sessionID, ExpiresAt: time.Now().Add(time.Hour)})
	require.NoError(t, err)
	return token
}

// activeSession simula que, en la próxima petición, la sesión del dispositivo
// sigue activa y el usuario tiene estos roles en la base de datos.
func (a testAPI) activeSession(d device, roles ...domain.Role) {
	a.sessions.EXPECT().ActiveRoles(mock.Anything, d.sessionID, d.userID).Return(roles, nil).Once()
}

// closedSession simula que la sesión del dispositivo ya fue cerrada.
func (a testAPI) closedSession(d device) {
	a.sessions.EXPECT().ActiveRoles(mock.Anything, d.sessionID, d.userID).Return(nil, domain.ErrSessionNotFound).Once()
}

func (a testAPI) do(method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	a.router.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &v), rec.Body.String())
	return v
}

func assertError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) dto.ErrorResponse {
	t.Helper()
	require.Equal(t, status, rec.Code, rec.Body.String())
	body := decode[dto.ErrorResponse](t, rec)
	assert.Equal(t, code, body.Error.Code)
	return body
}

// /healthz es el único health check: no requiere token ni consulta la base de datos.
func TestHealth(t *testing.T) {
	rec := newTestAPI(t).do(http.MethodGet, "/healthz", "", "")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"status":"ok"}`, rec.Body.String())
}

// RF-01.1: inicio de sesión con credenciales locales.
func TestLogin(t *testing.T) {
	api := newTestAPI(t)
	hash, err := api.hasher.Hash("secreta123")
	require.NoError(t, err)
	user := domain.User{
		ID: monitorID, FullName: "Ana Pérez", Email: "ana@uni.edu", PasswordHash: hash,
		Roles: []domain.Role{domain.RoleGraduateAssistant, domain.RoleMonitor},
	}
	api.repo.EXPECT().GetByEmail(mock.Anything, "ana@uni.edu").Return(user, nil)

	t.Run("credenciales válidas abren una sesión y entregan su token", func(t *testing.T) {
		api.sessions.EXPECT().Create(mock.Anything, mock.MatchedBy(func(s domain.Session) bool {
			return s.UserID == monitorID && s.UserAgent == userAgent && s.IP != ""
		})).RunAndReturn(func(_ context.Context, s domain.Session) (domain.Session, error) {
			s.ID = anaLaptop.sessionID
			return s, nil
		}).Once()

		rec := api.do(http.MethodPost, "/api/v1/auth/login", `{"email":"ANA@uni.edu","password":"secreta123"}`, "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		body := decode[dto.LoginResponse](t, rec)
		assert.Equal(t, "Bearer", body.TokenType)
		assert.Equal(t, []string{"ASISTENTE_GRADUADO", "MONITOR"}, body.User.Roles)
		assert.NotContains(t, rec.Body.String(), "password", "la respuesta nunca expone el hash")

		claims, err := api.tokens.Parse(body.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, monitorID, claims.UserID)
		assert.Equal(t, anaLaptop.sessionID, claims.SessionID)
	})

	t.Run("contraseña incorrecta", func(t *testing.T) {
		rec := api.do(http.MethodPost, "/api/v1/auth/login", `{"email":"ana@uni.edu","password":"otra-clave"}`, "")

		assertError(t, rec, http.StatusUnauthorized, "invalid_credentials")
		assert.NotEmpty(t, rec.Header().Get("WWW-Authenticate"))
	})

	t.Run("JSON mal formado", func(t *testing.T) {
		assertError(t, api.do(http.MethodPost, "/api/v1/auth/login", `{"email":`, ""), http.StatusBadRequest, "invalid_request")
	})

	t.Run("campos vacíos", func(t *testing.T) {
		body := assertError(t, api.do(http.MethodPost, "/api/v1/auth/login", `{}`, ""), http.StatusUnprocessableEntity, "validation_failed")
		assert.Contains(t, body.Error.Details, "email")
		assert.Contains(t, body.Error.Details, "password")
	})
}

// RF-01.2 y RF-01.3: control de acceso con los roles vigentes y usuarios con varios roles.
func TestAccessControl(t *testing.T) {
	api := newTestAPI(t)

	t.Run("sin token", func(t *testing.T) {
		assertError(t, api.do(http.MethodGet, "/api/v1/me", "", ""), http.StatusUnauthorized, "unauthenticated")
	})

	t.Run("token alterado", func(t *testing.T) {
		token := api.tokenFor(t, anaLaptop) + "x"
		assertError(t, api.do(http.MethodGet, "/api/v1/me", "", token), http.StatusUnauthorized, "unauthenticated")
	})

	t.Run("cualquier usuario autenticado consulta su identidad", func(t *testing.T) {
		roles := []domain.Role{domain.RoleGraduateAssistant, domain.RoleMonitor}
		api.activeSession(anaLaptop, roles...)
		api.repo.EXPECT().GetByID(mock.Anything, monitorID).
			Return(domain.User{ID: monitorID, Email: "ana@uni.edu", Roles: roles}, nil).Once()

		rec := api.do(http.MethodGet, "/api/v1/me", "", api.tokenFor(t, anaLaptop))

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, []string{"ASISTENTE_GRADUADO", "MONITOR"}, decode[dto.UserResponse](t, rec).Roles)
	})

	t.Run("un rol sin permisos recibe 403 en las rutas de administración", func(t *testing.T) {
		api.activeSession(anaLaptop, domain.RoleProfessor, domain.RoleMonitor)

		assertError(t, api.do(http.MethodGet, "/api/v1/admin/users", "", api.tokenFor(t, anaLaptop)), http.StatusForbidden, "forbidden")
	})

	t.Run("basta con tener ADMIN entre varios roles", func(t *testing.T) {
		api.activeSession(adminLaptop, domain.RoleProfessor, domain.RoleAdmin)
		api.repo.EXPECT().List(mock.Anything, mock.Anything).Return(ports.UserPage{}, nil).Once()

		assert.Equal(t, http.StatusOK, api.do(http.MethodGet, "/api/v1/admin/users", "", api.tokenFor(t, adminLaptop)).Code)
	})

	t.Run("un cambio de roles aplica en la siguiente petición con el mismo token", func(t *testing.T) {
		token := api.tokenFor(t, betoLaptop) // Beto inicia sesión siendo ADMIN.

		api.activeSession(betoLaptop, domain.RoleAdmin, domain.RoleProfessor)
		api.repo.EXPECT().List(mock.Anything, mock.Anything).Return(ports.UserPage{}, nil).Once()
		assert.Equal(t, http.StatusOK, api.do(http.MethodGet, "/api/v1/admin/users", "", token).Code)

		// Otro administrador le quita ADMIN: el mismo token ya no da acceso.
		api.activeSession(betoLaptop, domain.RoleProfessor)
		assertError(t, api.do(http.MethodGet, "/api/v1/admin/users", "", token), http.StatusForbidden, "forbidden")
	})

	t.Run("sesión cerrada o de un usuario eliminado", func(t *testing.T) {
		api.closedSession(betoLaptop)

		assertError(t, api.do(http.MethodGet, "/api/v1/me", "", api.tokenFor(t, betoLaptop)), http.StatusUnauthorized, "session_revoked")
	})

	t.Run("una falla de la base de datos responde 500, no 401", func(t *testing.T) {
		api.sessions.EXPECT().ActiveRoles(mock.Anything, betoLaptop.sessionID, betoID).
			Return(nil, errors.New("conexión rechazada")).Once()

		assertError(t, api.do(http.MethodGet, "/api/v1/me", "", api.tokenFor(t, betoLaptop)), http.StatusInternalServerError, "internal_error")
	})
}

// DT-03: sesiones por dispositivo, cierre de sesión y revocación por el administrador.
func TestSessions(t *testing.T) {
	api := newTestAPI(t)
	laptop, phone, admin := api.tokenFor(t, anaLaptop), api.tokenFor(t, anaPhone), api.tokenFor(t, adminLaptop)
	anaUser := domain.User{ID: monitorID, Email: "ana@uni.edu", Roles: []domain.Role{domain.RoleMonitor}}

	t.Run("cerrar sesión en el celular no cierra la del portátil", func(t *testing.T) {
		api.activeSession(anaPhone, domain.RoleMonitor)
		api.sessions.EXPECT().Revoke(mock.Anything, anaPhone.sessionID).Return(nil).Once()

		rec := api.do(http.MethodPost, "/api/v1/auth/logout", "", phone)
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())

		api.closedSession(anaPhone)
		assertError(t, api.do(http.MethodGet, "/api/v1/me", "", phone), http.StatusUnauthorized, "session_revoked")

		api.activeSession(anaLaptop, domain.RoleMonitor)
		api.repo.EXPECT().GetByID(mock.Anything, monitorID).Return(anaUser, nil).Once()
		assert.Equal(t, http.StatusOK, api.do(http.MethodGet, "/api/v1/me", "", laptop).Code)
	})

	t.Run("el administrador lista las sesiones activas de un usuario", func(t *testing.T) {
		api.activeSession(adminLaptop, domain.RoleAdmin)
		api.repo.EXPECT().GetByID(mock.Anything, monitorID).Return(anaUser, nil).Once()
		api.sessions.EXPECT().ListActive(mock.Anything, monitorID).Return([]domain.Session{
			{ID: anaLaptop.sessionID, UserID: monitorID, UserAgent: "Firefox", IP: "10.0.0.7"},
		}, nil).Once()

		rec := api.do(http.MethodGet, "/api/v1/admin/users/"+monitorID+"/sessions", "", admin)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		body := decode[dto.SessionListResponse](t, rec)
		require.Len(t, body.Items, 1)
		assert.Equal(t, "Firefox", body.Items[0].UserAgent)
	})

	t.Run("el administrador cierra todas las sesiones de un usuario", func(t *testing.T) {
		api.activeSession(adminLaptop, domain.RoleAdmin)
		api.repo.EXPECT().GetByID(mock.Anything, monitorID).Return(anaUser, nil).Once()
		api.sessions.EXPECT().RevokeAll(mock.Anything, monitorID).Return(1, nil).Once()

		rec := api.do(http.MethodDelete, "/api/v1/admin/users/"+monitorID+"/sessions", "", admin)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, 1, decode[dto.RevokeSessionsResponse](t, rec).Revoked)

		api.closedSession(anaLaptop)
		assertError(t, api.do(http.MethodGet, "/api/v1/me", "", laptop), http.StatusUnauthorized, "session_revoked")
	})

	t.Run("usuario inexistente", func(t *testing.T) {
		api.activeSession(adminLaptop, domain.RoleAdmin)
		api.repo.EXPECT().GetByID(mock.Anything, "no-existe").Return(domain.User{}, domain.ErrUserNotFound).Once()

		assertError(t, api.do(http.MethodDelete, "/api/v1/admin/users/no-existe/sessions", "", admin), http.StatusNotFound, "user_not_found")
	})

	t.Run("solo el administrador gestiona sesiones ajenas", func(t *testing.T) {
		api.activeSession(anaLaptop, domain.RoleMonitor)

		assertError(t, api.do(http.MethodDelete, "/api/v1/admin/users/"+adminID+"/sessions", "", laptop), http.StatusForbidden, "forbidden")
	})
}

// RF-02: gestión de usuarios por parte del administrador.
func TestAdminUsers(t *testing.T) {
	api := newTestAPI(t)
	token := api.tokenFor(t, adminLaptop)
	asAdmin := func() string {
		api.activeSession(adminLaptop, domain.RoleAdmin)
		return token
	}
	createdAt := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)

	t.Run("crea un usuario con varios roles", func(t *testing.T) {
		api.repo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(u domain.User) bool {
			return u.Email == "ana@uni.edu" && u.PasswordHash != "" && u.PasswordHash != "secreta123"
		})).RunAndReturn(func(_ context.Context, u domain.User) (domain.User, error) {
			u.ID, u.CreatedAt = monitorID, createdAt
			return u, nil
		}).Once()

		rec := api.do(http.MethodPost, "/api/v1/admin/users",
			`{"full_name":"Ana Pérez","email":"Ana@Uni.edu","password":"secreta123","roles":["MONITOR","ASISTENTE_GRADUADO"]}`, asAdmin())

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		assert.Equal(t, "/api/v1/admin/users/"+monitorID, rec.Header().Get("Location"))
		body := decode[dto.UserResponse](t, rec)
		assert.Equal(t, "ana@uni.edu", body.Email)
		assert.Equal(t, []string{"ASISTENTE_GRADUADO", "MONITOR"}, body.Roles)
		assert.NotContains(t, rec.Body.String(), "password")
	})

	t.Run("correo duplicado", func(t *testing.T) {
		api.repo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.User{}, domain.ErrEmailAlreadyExists).Once()

		rec := api.do(http.MethodPost, "/api/v1/admin/users",
			`{"full_name":"Ana","email":"ana@uni.edu","password":"secreta123","roles":["MONITOR"]}`, asAdmin())

		assertError(t, rec, http.StatusConflict, "email_already_exists")
	})

	t.Run("datos inválidos con detalle por campo", func(t *testing.T) {
		rec := api.do(http.MethodPost, "/api/v1/admin/users", `{"email":"x","password":"123","roles":["DECANO"]}`, asAdmin())

		body := assertError(t, rec, http.StatusUnprocessableEntity, "validation_failed")
		assert.Len(t, body.Error.Details, 4)
	})

	t.Run("lista con filtro y paginación", func(t *testing.T) {
		filter := ports.UserFilter{Role: domain.RoleMonitor, Limit: 10, Offset: 0}
		api.repo.EXPECT().List(mock.Anything, filter).Return(ports.UserPage{
			Users: []domain.User{{ID: monitorID, Email: "ana@uni.edu", Roles: []domain.Role{domain.RoleMonitor}}},
			Total: 1,
		}, nil).Once()

		rec := api.do(http.MethodGet, "/api/v1/admin/users?role=monitor&limit=10", "", asAdmin())

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		body := decode[dto.UserListResponse](t, rec)
		assert.Equal(t, 1, body.Total)
		assert.Equal(t, 10, body.Limit)
		require.Len(t, body.Items, 1)
		assert.Equal(t, monitorID, body.Items[0].ID)
	})

	t.Run("parámetros de listado mal formados o fuera de rango", func(t *testing.T) {
		assertError(t, api.do(http.MethodGet, "/api/v1/admin/users?limit=abc", "", asAdmin()), http.StatusBadRequest, "invalid_request")
		assertError(t, api.do(http.MethodGet, "/api/v1/admin/users?limit=999", "", asAdmin()), http.StatusUnprocessableEntity, "validation_failed")
	})

	t.Run("consulta un usuario inexistente", func(t *testing.T) {
		api.repo.EXPECT().GetByID(mock.Anything, "no-existe").Return(domain.User{}, domain.ErrUserNotFound).Once()

		assertError(t, api.do(http.MethodGet, "/api/v1/admin/users/no-existe", "", asAdmin()), http.StatusNotFound, "user_not_found")
	})

	t.Run("reemplaza los roles", func(t *testing.T) {
		api.repo.EXPECT().GetByID(mock.Anything, monitorID).
			Return(domain.User{ID: monitorID, Roles: []domain.Role{domain.RoleMonitor}}, nil).Once()
		api.repo.EXPECT().ReplaceRoles(mock.Anything, monitorID, []domain.Role{domain.RoleGraduateAssistant}).Return(nil).Once()

		rec := api.do(http.MethodPut, "/api/v1/admin/users/"+monitorID+"/roles", `{"roles":["ASISTENTE_GRADUADO"]}`, asAdmin())

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, []string{"ASISTENTE_GRADUADO"}, decode[dto.UserResponse](t, rec).Roles)
	})

	t.Run("no deja el sistema sin administradores", func(t *testing.T) {
		api.repo.EXPECT().GetByID(mock.Anything, adminID).
			Return(domain.User{ID: adminID, Roles: []domain.Role{domain.RoleAdmin}}, nil).Once()
		api.repo.EXPECT().CountWithRole(mock.Anything, domain.RoleAdmin).Return(1, nil).Once()

		rec := api.do(http.MethodPut, "/api/v1/admin/users/"+adminID+"/roles", `{"roles":["PROFESOR"]}`, asAdmin())

		assertError(t, rec, http.StatusConflict, "last_admin")
	})
}

func TestUnknownRoute(t *testing.T) {
	api := newTestAPI(t)
	assertError(t, api.do(http.MethodGet, "/api/v1/no-existe", "", ""), http.StatusNotFound, "route_not_found")
}
