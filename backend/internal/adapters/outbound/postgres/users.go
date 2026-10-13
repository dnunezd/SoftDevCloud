package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
	"github.com/dnunezd/SoftDevCloud/backend/internal/ports"
)

const (
	uniqueViolation     = "23505"
	usersEmailUniqueKey = "users_email_key"

	selectUser = `
SELECT u.id::text, u.full_name, u.email, u.password_hash, u.created_at,
       COALESCE(array_agg(r.role ORDER BY r.role) FILTER (WHERE r.role IS NOT NULL), '{}')
FROM users u
LEFT JOIN user_roles r ON r.user_id = u.id
`
	selectUserByID    = selectUser + `WHERE u.id = $1 GROUP BY u.id`
	selectUserByEmail = selectUser + `WHERE u.email = $1 GROUP BY u.id`
	selectUserPage    = selectUser + `
WHERE $1::text = '' OR EXISTS (SELECT 1 FROM user_roles f WHERE f.user_id = u.id AND f.role = $1::text)
GROUP BY u.id
ORDER BY u.full_name, u.id
LIMIT $2 OFFSET $3`
	countUsers = `
SELECT count(*) FROM users u
WHERE $1::text = '' OR EXISTS (SELECT 1 FROM user_roles f WHERE f.user_id = u.id AND f.role = $1::text)`
	insertUser = `
INSERT INTO users (full_name, email, password_hash) VALUES ($1, $2, $3)
RETURNING id::text, created_at`
	insertUserRoles = `INSERT INTO user_roles (user_id, role) SELECT $1::uuid, unnest($2::text[])`
	touchUser       = `UPDATE users SET updated_at = now() WHERE id = $1`
	deleteUserRoles = `DELETE FROM user_roles WHERE user_id = $1`
	countWithRole   = `SELECT count(*) FROM user_roles WHERE role = $1`
)

// UserRepository implementa ports.UserRepository sobre PostgreSQL.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository crea el repositorio.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create inserta el usuario y sus roles en una misma transacción.
func (r *UserRepository) Create(ctx context.Context, user domain.User) (domain.User, error) {
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, insertUser, user.FullName, user.Email, user.PasswordHash).
			Scan(&user.ID, &user.CreatedAt)
		if isUniqueViolation(err, usersEmailUniqueKey) {
			return domain.ErrEmailAlreadyExists
		}
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, insertUserRoles, user.ID, roleNames(user.Roles))
		return err
	})
	if errors.Is(err, domain.ErrEmailAlreadyExists) {
		return domain.User{}, err
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("insertar usuario: %w", err)
	}
	return user, nil
}

// GetByID busca un usuario por su ID. Un ID con formato inválido se trata como inexistente.
func (r *UserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	if !isUUID(id) {
		return domain.User{}, domain.ErrUserNotFound
	}
	return r.getOne(ctx, selectUserByID, id)
}

// GetByEmail busca un usuario por su correo normalizado.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	return r.getOne(ctx, selectUserByEmail, email)
}

// List devuelve una página de usuarios ordenada por nombre y el total de coincidencias.
func (r *UserRepository) List(ctx context.Context, filter ports.UserFilter) (ports.UserPage, error) {
	var page ports.UserPage
	role := string(filter.Role)
	if err := r.pool.QueryRow(ctx, countUsers, role).Scan(&page.Total); err != nil {
		return ports.UserPage{}, fmt.Errorf("contar usuarios: %w", err)
	}

	rows, err := r.pool.Query(ctx, selectUserPage, role, filter.Limit, filter.Offset)
	if err != nil {
		return ports.UserPage{}, fmt.Errorf("listar usuarios: %w", err)
	}
	page.Users, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.User, error) {
		return scanUser(row)
	})
	if err != nil {
		return ports.UserPage{}, fmt.Errorf("leer usuarios: %w", err)
	}
	return page, nil
}

// ReplaceRoles reemplaza los roles del usuario en una transacción. El UPDATE
// bloquea la fila del usuario mientras se cambian sus roles.
func (r *UserRepository) ReplaceRoles(ctx context.Context, id string, roles []domain.Role) error {
	if !isUUID(id) {
		return domain.ErrUserNotFound
	}
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, touchUser, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrUserNotFound
		}
		if _, err := tx.Exec(ctx, deleteUserRoles, id); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, insertUserRoles, id, roleNames(roles))
		return err
	})
	if errors.Is(err, domain.ErrUserNotFound) {
		return err
	}
	if err != nil {
		return fmt.Errorf("reemplazar roles: %w", err)
	}
	return nil
}

// CountWithRole cuenta los usuarios que tienen el rol.
func (r *UserRepository) CountWithRole(ctx context.Context, role domain.Role) (int, error) {
	var n int
	if err := r.pool.QueryRow(ctx, countWithRole, string(role)).Scan(&n); err != nil {
		return 0, fmt.Errorf("contar usuarios con rol: %w", err)
	}
	return n, nil
}

func (r *UserRepository) getOne(ctx context.Context, query string, arg string) (domain.User, error) {
	user, err := scanUser(r.pool.QueryRow(ctx, query, arg))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("consultar usuario: %w", err)
	}
	return user, nil
}

func scanUser(row pgx.Row) (domain.User, error) {
	var (
		user  domain.User
		roles []string
	)
	if err := row.Scan(&user.ID, &user.FullName, &user.Email, &user.PasswordHash, &user.CreatedAt, &roles); err != nil {
		return domain.User{}, err
	}
	user.Roles = toRoles(roles)
	return user, nil
}

func toRoles(names []string) []domain.Role {
	roles := make([]domain.Role, len(names))
	for i, name := range names {
		roles[i] = domain.Role(name)
	}
	return roles
}

func roleNames(roles []domain.Role) []string {
	names := make([]string, len(roles))
	for i, role := range roles {
		names[i] = string(role)
	}
	return names
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolation && pgErr.ConstraintName == constraint
}

// isUUID valida el formato canónico 8-4-4-4-12 sin depender de librerías externas.
func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !isHex(c) {
				return false
			}
		}
	}
	return true
}

func isHex(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
