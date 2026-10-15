package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/inbound/rest/dto"
	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/inbound/rest/middleware"
	"github.com/dnunezd/SoftDevCloud/backend/internal/app/workspaces"
	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

// Workspaces atiende los cursos y proyectos (RF-03).
type Workspaces struct {
	workspaces *workspaces.Service
}

// NewWorkspaces crea los handlers de cursos y proyectos.
func NewWorkspaces(workspaceService *workspaces.Service) *Workspaces {
	return &Workspaces{workspaces: workspaceService}
}

// Create atiende POST /api/v1/workspaces (RF-03.1, RF-03.3).
func (h *Workspaces) Create(c *gin.Context) {
	principal, ok := middleware.PrincipalFrom(c)
	if !ok {
		fail(c, domain.ErrUnauthenticated)
		return
	}
	var req dto.CreateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, middleware.InvalidRequest(err))
		return
	}
	workspace, err := h.workspaces.Create(c.Request.Context(), principal, req.ToInput())
	if err != nil {
		fail(c, err)
		return
	}
	c.Header("Location", "/api/v1/workspaces/"+workspace.ID)
	c.JSON(http.StatusCreated, dto.NewWorkspaceResponse(workspace))
}

// List atiende GET /api/v1/workspaces?period_id=&status=&limit=&offset= (RF-03.4, RF-03.6).
func (h *Workspaces) List(c *gin.Context) {
	principal, ok := middleware.PrincipalFrom(c)
	if !ok {
		fail(c, domain.ErrUnauthenticated)
		return
	}
	var query dto.ListWorkspacesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		fail(c, middleware.InvalidRequest(err))
		return
	}
	page, err := h.workspaces.List(c.Request.Context(), principal, query.ToInput())
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewWorkspaceListResponse(page))
}

// Get atiende GET /api/v1/workspaces/{id} (RF-03.4, RF-03.6).
func (h *Workspaces) Get(c *gin.Context) {
	principal, ok := middleware.PrincipalFrom(c)
	if !ok {
		fail(c, domain.ErrUnauthenticated)
		return
	}
	workspace, err := h.workspaces.Get(c.Request.Context(), principal, c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewWorkspaceResponse(workspace))
}

// Close atiende POST /api/v1/workspaces/{id}/close (RF-03.5).
func (h *Workspaces) Close(c *gin.Context) {
	principal, ok := middleware.PrincipalFrom(c)
	if !ok {
		fail(c, domain.ErrUnauthenticated)
		return
	}
	workspace, err := h.workspaces.Close(c.Request.Context(), principal, c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewWorkspaceResponse(workspace))
}
