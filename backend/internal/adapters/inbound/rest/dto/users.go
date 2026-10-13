package dto

import (
	"time"

	"github.com/dnunezd/SoftDevCloud/backend/internal/app/users"
	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

// CreateUserRequest es el cuerpo de POST /api/v1/admin/users.
type CreateUserRequest struct {
	FullName string   `json:"full_name"`
	Email    string   `json:"email"`
	Password string   `json:"password"`
	Roles    []string `json:"roles"`
}

// ToInput convierte la petición en la entrada del caso de uso.
func (r CreateUserRequest) ToInput() users.CreateInput {
	return users.CreateInput{FullName: r.FullName, Email: r.Email, Password: r.Password, Roles: r.Roles}
}

// SetRolesRequest es el cuerpo de PUT /api/v1/admin/users/{id}/roles.
type SetRolesRequest struct {
	Roles []string `json:"roles"`
}

// ListUsersQuery son los parámetros de GET /api/v1/admin/users.
type ListUsersQuery struct {
	Role   string `form:"role"`
	Limit  int    `form:"limit"`
	Offset int    `form:"offset"`
}

// ToInput convierte los parámetros en la entrada del caso de uso.
func (q ListUsersQuery) ToInput() users.ListInput {
	return users.ListInput{Role: q.Role, Limit: q.Limit, Offset: q.Offset}
}

// UserResponse es la representación pública de un usuario.
type UserResponse struct {
	ID        string    `json:"id"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
}

// NewUserResponse convierte un usuario de dominio.
func NewUserResponse(u domain.User) UserResponse {
	roles := make([]string, len(u.Roles))
	for i, role := range u.Roles {
		roles[i] = string(role)
	}
	return UserResponse{ID: u.ID, FullName: u.FullName, Email: u.Email, Roles: roles, CreatedAt: u.CreatedAt}
}

// UserListResponse es una página del listado de usuarios.
type UserListResponse struct {
	Items  []UserResponse `json:"items"`
	Total  int            `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

// NewUserListResponse convierte una página del caso de uso.
func NewUserListResponse(p users.Page) UserListResponse {
	items := make([]UserResponse, len(p.Users))
	for i, u := range p.Users {
		items[i] = NewUserResponse(u)
	}
	return UserListResponse{Items: items, Total: p.Total, Limit: p.Limit, Offset: p.Offset}
}
