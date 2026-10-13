// Package users implementa la gestión de usuarios por parte del administrador:
// creación, asignación de roles y consulta global (RF-02, RF-15.2).
package users

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
	"github.com/dnunezd/SoftDevCloud/backend/internal/ports"
)

// Límites de paginación del listado de usuarios.
const (
	DefaultPageSize = 50
	MaxPageSize     = 200
)

// Service gestiona usuarios y sus roles.
type Service struct {
	repo   ports.UserRepository
	hasher ports.PasswordHasher
}

// NewService crea el servicio de usuarios.
func NewService(repo ports.UserRepository, hasher ports.PasswordHasher) *Service {
	return &Service{repo: repo, hasher: hasher}
}

// CreateInput son los datos para crear un usuario.
type CreateInput struct {
	FullName string
	Email    string
	Password string
	Roles    []string
}

// Create registra un usuario con uno o varios roles (RF-02.1, RF-02.2).
func (s *Service) Create(ctx context.Context, in CreateInput) (domain.User, error) {
	user, err := domain.UserRegistration{
		FullName: in.FullName,
		Email:    in.Email,
		Password: in.Password,
		Roles:    in.Roles,
	}.Validate()
	if err != nil {
		return domain.User{}, err
	}

	hash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return domain.User{}, fmt.Errorf("calcular hash de contraseña: %w", err)
	}
	user.PasswordHash = hash

	created, err := s.repo.Create(ctx, user)
	if err != nil {
		return domain.User{}, fmt.Errorf("crear usuario: %w", err)
	}
	return created, nil
}

// ListInput son los criterios del listado. Los valores cero aplican los
// valores por defecto.
type ListInput struct {
	Role   string
	Limit  int
	Offset int
}

// Page es una página del listado de usuarios.
type Page struct {
	Users  []domain.User
	Total  int
	Limit  int
	Offset int
}

// List consulta los usuarios del sistema, opcionalmente filtrados por rol (RF-02.3).
func (s *Service) List(ctx context.Context, in ListInput) (Page, error) {
	filter := ports.UserFilter{Limit: in.Limit, Offset: in.Offset}
	fields := map[string]string{}

	if in.Role != "" {
		roles, err := domain.ParseRoles([]string{in.Role})
		if err != nil {
			fields["role"] = "rol desconocido: " + in.Role
		} else {
			filter.Role = roles[0]
		}
	}
	if filter.Limit == 0 {
		filter.Limit = DefaultPageSize
	}
	if filter.Limit < 1 || filter.Limit > MaxPageSize {
		fields["limit"] = fmt.Sprintf("debe estar entre 1 y %d", MaxPageSize)
	}
	if filter.Offset < 0 {
		fields["offset"] = "no puede ser negativo"
	}
	if len(fields) > 0 {
		return Page{}, domain.NewValidationError(fields)
	}

	result, err := s.repo.List(ctx, filter)
	if err != nil {
		return Page{}, fmt.Errorf("listar usuarios: %w", err)
	}
	return Page{Users: result.Users, Total: result.Total, Limit: filter.Limit, Offset: filter.Offset}, nil
}

// Get consulta un usuario por su ID.
func (s *Service) Get(ctx context.Context, id string) (domain.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.User{}, fmt.Errorf("consultar usuario: %w", err)
	}
	return user, nil
}

// SetRoles reemplaza los roles de un usuario (RF-02.2). Impide dejar el sistema
// sin administradores.
func (s *Service) SetRoles(ctx context.Context, id string, roles []string) (domain.User, error) {
	parsed, err := domain.ParseRoles(roles)
	if err != nil {
		return domain.User{}, err
	}

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.User{}, fmt.Errorf("consultar usuario: %w", err)
	}

	if user.HasRole(domain.RoleAdmin) && !slices.Contains(parsed, domain.RoleAdmin) {
		admins, err := s.repo.CountWithRole(ctx, domain.RoleAdmin)
		if err != nil {
			return domain.User{}, fmt.Errorf("contar administradores: %w", err)
		}
		if admins <= 1 {
			return domain.User{}, domain.ErrLastAdmin
		}
	}

	if err := s.repo.ReplaceRoles(ctx, id, parsed); err != nil {
		return domain.User{}, fmt.Errorf("reemplazar roles: %w", err)
	}
	user.Roles = parsed
	return user, nil
}

// EnsureAdmin crea el administrador inicial si todavía no existe ninguno.
// Devuelve true si lo creó. Es seguro ejecutarlo en varias instancias a la vez.
func (s *Service) EnsureAdmin(ctx context.Context, fullName, email, password string) (bool, error) {
	admins, err := s.repo.CountWithRole(ctx, domain.RoleAdmin)
	if err != nil {
		return false, fmt.Errorf("contar administradores: %w", err)
	}
	if admins > 0 {
		return false, nil
	}

	_, err = s.Create(ctx, CreateInput{
		FullName: fullName,
		Email:    email,
		Password: password,
		Roles:    []string{string(domain.RoleAdmin)},
	})
	if errors.Is(err, domain.ErrEmailAlreadyExists) {
		// Otra instancia lo creó en paralelo.
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
