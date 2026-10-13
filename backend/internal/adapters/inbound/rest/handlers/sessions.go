package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/inbound/rest/dto"
	"github.com/dnunezd/SoftDevCloud/backend/internal/app/auth"
)

// Sessions atiende la gestión de sesiones de un usuario por parte del administrador.
type Sessions struct {
	auth *auth.Service
}

// NewSessions crea los handlers de sesiones.
func NewSessions(authService *auth.Service) *Sessions {
	return &Sessions{auth: authService}
}

// List atiende GET /api/v1/admin/users/{id}/sessions: sesiones activas del usuario.
func (h *Sessions) List(c *gin.Context) {
	sessions, err := h.auth.ListSessions(c.Request.Context(), c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewSessionListResponse(sessions))
}

// RevokeAll atiende DELETE /api/v1/admin/users/{id}/sessions: cierra todas las
// sesiones del usuario, por ejemplo si su token fue robado.
func (h *Sessions) RevokeAll(c *gin.Context) {
	revoked, err := h.auth.RevokeSessions(c.Request.Context(), c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.RevokeSessionsResponse{Revoked: revoked})
}
