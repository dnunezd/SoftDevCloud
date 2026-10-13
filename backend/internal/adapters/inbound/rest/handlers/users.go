package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/inbound/rest/dto"
	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/inbound/rest/middleware"
	"github.com/dnunezd/SoftDevCloud/backend/internal/app/users"
)

// Users atiende la gestión de usuarios por parte del administrador (RF-02, RF-15.2).
type Users struct {
	users *users.Service
}

// NewUsers crea los handlers de usuarios.
func NewUsers(userService *users.Service) *Users {
	return &Users{users: userService}
}

// Create atiende POST /api/v1/admin/users (RF-02.1, RF-02.2).
func (h *Users) Create(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, middleware.InvalidRequest(err))
		return
	}
	user, err := h.users.Create(c.Request.Context(), req.ToInput())
	if err != nil {
		fail(c, err)
		return
	}
	c.Header("Location", "/api/v1/admin/users/"+user.ID)
	c.JSON(http.StatusCreated, dto.NewUserResponse(user))
}

// List atiende GET /api/v1/admin/users?role=&limit=&offset= (RF-02.3).
func (h *Users) List(c *gin.Context) {
	var query dto.ListUsersQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		fail(c, middleware.InvalidRequest(err))
		return
	}
	page, err := h.users.List(c.Request.Context(), query.ToInput())
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewUserListResponse(page))
}

// Get atiende GET /api/v1/admin/users/{id} (RF-02.3).
func (h *Users) Get(c *gin.Context) {
	user, err := h.users.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewUserResponse(user))
}

// SetRoles atiende PUT /api/v1/admin/users/{id}/roles (RF-02.2).
func (h *Users) SetRoles(c *gin.Context) {
	var req dto.SetRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, middleware.InvalidRequest(err))
		return
	}
	user, err := h.users.SetRoles(c.Request.Context(), c.Param("id"), req.Roles)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewUserResponse(user))
}
