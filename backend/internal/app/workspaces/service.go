// Package workspaces implementa la gestión de cursos y proyectos: creación
// dentro de la ventana del período, consulta, cierre y consulta en modo
// solo lectura (RF-03).
package workspaces

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dnunezd/SoftDevCloud/backend/internal/app/access"
	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
	"github.com/dnunezd/SoftDevCloud/backend/internal/ports"
)

// Límites de paginación del listado de espacios.
const (
	DefaultPageSize = 50
	MaxPageSize     = 200
)

// Service gestiona cursos y proyectos.
type Service struct {
	workspaces ports.WorkspaceRepository
	periods    ports.PeriodRepository
	clock      ports.Clock
	loc        *time.Location
}

// NewService crea el servicio de espacios. loc es la zona horaria académica
// (APP_TIMEZONE) con la que se decide si la ventana de creación sigue abierta.
func NewService(workspaces ports.WorkspaceRepository, periods ports.PeriodRepository, clock ports.Clock, loc *time.Location) *Service {
	return &Service{workspaces: workspaces, periods: periods, clock: clock, loc: loc}
}

// CreateInput son los datos para crear un curso o proyecto. Las fechas llegan
// como texto AAAA-MM-DD.
type CreateInput struct {
	Type         string
	Name         string
	PeriodID     string
	StartDate    string
	EndDate      string
	Observations string
}

// Create registra un curso o proyecto del profesor autenticado, que queda como
// profesor responsable (RF-03.1, RF-03.3). Solo se permite mientras el período
// esté activo y dentro de su ventana de creación (RF-03.2, RN-07).
func (s *Service) Create(ctx context.Context, who access.Principal, in CreateInput) (domain.Workspace, error) {
	if !who.HasRole(domain.RoleProfessor) {
		return domain.Workspace{}, domain.ErrForbidden
	}

	fields := map[string]string{}
	workspace := domain.Workspace{
		PeriodID:     strings.TrimSpace(in.PeriodID),
		ProfessorID:  who.UserID,
		Type:         domain.WorkspaceType(strings.ToUpper(strings.TrimSpace(in.Type))),
		Name:         strings.TrimSpace(in.Name),
		Observations: strings.TrimSpace(in.Observations),
		Status:       domain.WorkspaceActive,
	}
	workspace.StartDate = domain.ParseDateField(fields, "start_date", in.StartDate)
	workspace.EndDate = domain.ParseDateField(fields, "end_date", in.EndDate)
	if len(fields) > 0 {
		// Con una fecha inválida no tiene sentido comparar inicio y cierre.
		workspace.StartDate, workspace.EndDate = time.Time{}, time.Time{}
	}
	if err := workspace.Validate(); err != nil {
		var derr *domain.Error
		if errors.As(err, &derr) {
			for field, message := range derr.Fields {
				fields[field] = message
			}
		}
	}
	if workspace.PeriodID == "" {
		fields["period_id"] = "es obligatorio"
	}
	if len(fields) > 0 {
		return domain.Workspace{}, domain.NewValidationError(fields)
	}

	period, err := s.periods.GetByID(ctx, workspace.PeriodID)
	if errors.Is(err, domain.ErrPeriodNotFound) {
		return domain.Workspace{}, domain.NewValidationError(map[string]string{"period_id": "el período no existe"})
	}
	if err != nil {
		return domain.Workspace{}, fmt.Errorf("consultar período: %w", err)
	}
	if !period.AcceptsWorkspaceCreation(domain.Today(s.clock.Now(), s.loc)) {
		return domain.Workspace{}, domain.ErrCreationWindowClosed
	}

	created, err := s.workspaces.Create(ctx, workspace)
	if err != nil {
		return domain.Workspace{}, fmt.Errorf("crear curso o proyecto: %w", err)
	}
	return created, nil
}

// ListInput son los criterios del listado. Los valores cero aplican los
// valores por defecto.
type ListInput struct {
	PeriodID string
	Status   string
	Limit    int
	Offset   int
}

// Page es una página del listado de espacios.
type Page struct {
	Workspaces []domain.Workspace
	Total      int
	Limit      int
	Offset     int
}

// List consulta los espacios visibles para la identidad: el profesor ve solo
// los suyos y el administrador, todos (RF-03.4, RF-12.1, RN-05). Incluye los
// cerrados, que se consultan en modo de solo lectura (RF-03.6).
func (s *Service) List(ctx context.Context, who access.Principal, in ListInput) (Page, error) {
	if !who.HasAnyRole(domain.RoleProfessor, domain.RoleAdmin) {
		return Page{}, domain.ErrForbidden
	}

	filter := ports.WorkspaceFilter{PeriodID: in.PeriodID, Limit: in.Limit, Offset: in.Offset}
	fields := map[string]string{}
	if in.Status != "" {
		status := domain.WorkspaceStatus(strings.ToUpper(in.Status))
		if status != domain.WorkspaceActive && status != domain.WorkspaceClosed {
			fields["status"] = "debe ser ACTIVO o CERRADO"
		}
		filter.Status = status
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
	if !who.HasRole(domain.RoleAdmin) {
		filter.ProfessorID = who.UserID
	}

	result, err := s.workspaces.List(ctx, filter)
	if err != nil {
		return Page{}, fmt.Errorf("listar cursos y proyectos: %w", err)
	}
	return Page{Workspaces: result.Workspaces, Total: result.Total, Limit: filter.Limit, Offset: filter.Offset}, nil
}

// Get consulta un espacio. Si la identidad no puede verlo responde igual que
// si no existiera, para no revelar los espacios de otros profesores (RF-12.1).
func (s *Service) Get(ctx context.Context, who access.Principal, id string) (domain.Workspace, error) {
	return s.visible(ctx, who, id)
}

// Close cierra un curso o proyecto: desde entonces queda en modo de solo
// lectura y no admite nuevos registros operativos (RF-03.5, RF-03.7, RN-09).
// Solo lo cierra su profesor responsable.
func (s *Service) Close(ctx context.Context, who access.Principal, id string) (domain.Workspace, error) {
	workspace, err := s.visible(ctx, who, id)
	if err != nil {
		return domain.Workspace{}, err
	}
	if !who.CanManageWorkspace(workspace) {
		return domain.Workspace{}, domain.ErrForbidden
	}

	now := s.clock.Now()
	if err := workspace.Close(now); err != nil {
		return domain.Workspace{}, err
	}
	if err := s.workspaces.MarkClosed(ctx, id, now); err != nil {
		return domain.Workspace{}, fmt.Errorf("cerrar curso o proyecto: %w", err)
	}
	return workspace, nil
}

// visible carga el espacio y comprueba que la identidad pueda verlo.
func (s *Service) visible(ctx context.Context, who access.Principal, id string) (domain.Workspace, error) {
	workspace, err := s.workspaces.GetByID(ctx, id)
	if err != nil {
		return domain.Workspace{}, fmt.Errorf("consultar curso o proyecto: %w", err)
	}
	if !who.CanViewWorkspace(workspace) {
		return domain.Workspace{}, domain.ErrWorkspaceNotFound
	}
	return workspace, nil
}
