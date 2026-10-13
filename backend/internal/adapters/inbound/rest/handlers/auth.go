package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/inbound/rest/dto"
	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/inbound/rest/middleware"
	"github.com/dnunezd/SoftDevCloud/backend/internal/app/auth"
	"github.com/dnunezd/SoftDevCloud/backend/internal/app/users"
	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

// Auth atiende el inicio de sesión y la consulta de la identidad propia (RF-01).
type Auth struct {
	auth  *auth.Service
	users *users.Service
}

// NewAuth crea los handlers de autenticación.
func NewAuth(authService *auth.Service, userService *users.Service) *Auth {
	return &Auth{auth: authService, users: userService}
}

// Login atiende POST /api/v1/auth/login (RF-01.1).
func (h *Auth) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, middleware.InvalidRequest(err))
		return
	}
	client := auth.Client{UserAgent: c.Request.UserAgent(), IP: c.ClientIP()}
	result, err := h.auth.Login(c.Request.Context(), req.Email, req.Password, client)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewLoginResponse(result))
}

// Logout atiende POST /api/v1/auth/logout: cierra solo la sesión de este
// dispositivo. Las demás sesiones del usuario siguen activas.
func (h *Auth) Logout(c *gin.Context) {
	principal, ok := middleware.PrincipalFrom(c)
	if !ok {
		fail(c, domain.ErrUnauthenticated)
		return
	}
	if err := h.auth.Logout(c.Request.Context(), principal); err != nil {
		fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Me atiende GET /api/v1/me: datos y roles del usuario autenticado (RF-01.3).
func (h *Auth) Me(c *gin.Context) {
	principal, ok := middleware.PrincipalFrom(c)
	if !ok {
		fail(c, domain.ErrUnauthenticated)
		return
	}
	user, err := h.users.Get(c.Request.Context(), principal.UserID)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewUserResponse(user))
}
