package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
	"github.com/dnunezd/SoftDevCloud/backend/internal/ports"
)

const (
	selectWorkspace = `
SELECT id::text, period_id::text, professor_id::text, type, name, start_date, end_date, observations, status, closed_at
FROM workspaces
`
	selectWorkspaceByID = selectWorkspace + `WHERE id = $1`
	workspaceFilter     = `
WHERE ($1::text = '' OR professor_id = $1::uuid)
  AND ($2::text = '' OR period_id = $2::uuid)
  AND ($3::text = '' OR status = $3::text)`
	selectWorkspacePage = selectWorkspace + workspaceFilter + `
ORDER BY start_date DESC, name, id
LIMIT $4 OFFSET $5`
	countWorkspaces = `SELECT count(*) FROM workspaces` + workspaceFilter
	insertWorkspace = `
INSERT INTO workspaces (type, name, period_id, professor_id, start_date, end_date, observations, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id::text`
	closeWorkspace  = `UPDATE workspaces SET status = 'CERRADO', closed_at = $2 WHERE id = $1 AND status = 'ACTIVO'`
	workspaceExists = `SELECT EXISTS (SELECT 1 FROM workspaces WHERE id = $1)`
)

// WorkspaceRepository implementa ports.WorkspaceRepository sobre PostgreSQL.
type WorkspaceRepository struct {
	pool *pgxpool.Pool
}

// NewWorkspaceRepository crea el repositorio.
func NewWorkspaceRepository(pool *pgxpool.Pool) *WorkspaceRepository {
	return &WorkspaceRepository{pool: pool}
}

// Create inserta el espacio.
func (r *WorkspaceRepository) Create(ctx context.Context, workspace domain.Workspace) (domain.Workspace, error) {
	err := r.pool.QueryRow(ctx, insertWorkspace,
		string(workspace.Type), workspace.Name, workspace.PeriodID, workspace.ProfessorID,
		workspace.StartDate, workspace.EndDate, workspace.Observations, string(workspace.Status)).Scan(&workspace.ID)
	if err != nil {
		return domain.Workspace{}, fmt.Errorf("insertar curso o proyecto: %w", err)
	}
	return workspace, nil
}

// GetByID busca un espacio por su ID. Un ID con formato inválido se trata como inexistente.
func (r *WorkspaceRepository) GetByID(ctx context.Context, id string) (domain.Workspace, error) {
	if !isUUID(id) {
		return domain.Workspace{}, domain.ErrWorkspaceNotFound
	}
	workspace, err := scanWorkspace(r.pool.QueryRow(ctx, selectWorkspaceByID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Workspace{}, domain.ErrWorkspaceNotFound
	}
	if err != nil {
		return domain.Workspace{}, fmt.Errorf("consultar curso o proyecto: %w", err)
	}
	return workspace, nil
}

// List devuelve una página de espacios, de los que inician más tarde a los más
// antiguos, y el total de coincidencias. Un filtro con ID mal formado no
// coincide con nada.
func (r *WorkspaceRepository) List(ctx context.Context, filter ports.WorkspaceFilter) (ports.WorkspacePage, error) {
	if (filter.ProfessorID != "" && !isUUID(filter.ProfessorID)) || (filter.PeriodID != "" && !isUUID(filter.PeriodID)) {
		return ports.WorkspacePage{}, nil
	}
	status := string(filter.Status)

	var page ports.WorkspacePage
	if err := r.pool.QueryRow(ctx, countWorkspaces, filter.ProfessorID, filter.PeriodID, status).Scan(&page.Total); err != nil {
		return ports.WorkspacePage{}, fmt.Errorf("contar cursos y proyectos: %w", err)
	}
	rows, err := r.pool.Query(ctx, selectWorkspacePage, filter.ProfessorID, filter.PeriodID, status, filter.Limit, filter.Offset)
	if err != nil {
		return ports.WorkspacePage{}, fmt.Errorf("listar cursos y proyectos: %w", err)
	}
	page.Workspaces, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Workspace, error) {
		return scanWorkspace(row)
	})
	if err != nil {
		return ports.WorkspacePage{}, fmt.Errorf("leer cursos y proyectos: %w", err)
	}
	return page, nil
}

// MarkClosed cierra el espacio con un UPDATE condicionado al estado ACTIVO, así
// dos cierres simultáneos no se pisan ni cambian la fecha de cierre original.
func (r *WorkspaceRepository) MarkClosed(ctx context.Context, id string, closedAt time.Time) error {
	if !isUUID(id) {
		return domain.ErrWorkspaceNotFound
	}
	tag, err := r.pool.Exec(ctx, closeWorkspace, id, closedAt)
	if err != nil {
		return fmt.Errorf("cerrar curso o proyecto: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return nil
	}
	var exists bool
	if err := r.pool.QueryRow(ctx, workspaceExists, id).Scan(&exists); err != nil {
		return fmt.Errorf("consultar curso o proyecto: %w", err)
	}
	if !exists {
		return domain.ErrWorkspaceNotFound
	}
	return domain.ErrWorkspaceClosed
}

func scanWorkspace(row pgx.Row) (domain.Workspace, error) {
	var (
		workspace    domain.Workspace
		kind, status string
	)
	if err := row.Scan(&workspace.ID, &workspace.PeriodID, &workspace.ProfessorID, &kind, &workspace.Name,
		&workspace.StartDate, &workspace.EndDate, &workspace.Observations, &status, &workspace.ClosedAt); err != nil {
		return domain.Workspace{}, err
	}
	workspace.Type = domain.WorkspaceType(kind)
	workspace.Status = domain.WorkspaceStatus(status)
	return workspace, nil
}
