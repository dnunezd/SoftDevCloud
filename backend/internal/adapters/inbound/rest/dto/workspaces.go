package dto

import (
	"time"

	"github.com/dnunezd/SoftDevCloud/backend/internal/app/workspaces"
	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

// CreateWorkspaceRequest es el cuerpo de POST /api/v1/workspaces. type es CURSO
// o PROYECTO y las fechas van en formato AAAA-MM-DD.
type CreateWorkspaceRequest struct {
	Type         string `json:"type"`
	Name         string `json:"name"`
	PeriodID     string `json:"period_id"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
	Observations string `json:"observations"`
}

// ToInput convierte la petición en la entrada del caso de uso.
func (r CreateWorkspaceRequest) ToInput() workspaces.CreateInput {
	return workspaces.CreateInput{
		Type: r.Type, Name: r.Name, PeriodID: r.PeriodID,
		StartDate: r.StartDate, EndDate: r.EndDate, Observations: r.Observations,
	}
}

// ListWorkspacesQuery son los parámetros de GET /api/v1/workspaces.
type ListWorkspacesQuery struct {
	PeriodID string `form:"period_id"`
	Status   string `form:"status"`
	Limit    int    `form:"limit"`
	Offset   int    `form:"offset"`
}

// ToInput convierte los parámetros en la entrada del caso de uso.
func (q ListWorkspacesQuery) ToInput() workspaces.ListInput {
	return workspaces.ListInput{PeriodID: q.PeriodID, Status: q.Status, Limit: q.Limit, Offset: q.Offset}
}

// WorkspaceResponse es la representación pública de un curso o proyecto.
// closed_at solo aparece cuando el espacio está cerrado.
type WorkspaceResponse struct {
	ID           string     `json:"id"`
	Type         string     `json:"type"`
	Name         string     `json:"name"`
	PeriodID     string     `json:"period_id"`
	ProfessorID  string     `json:"professor_id"`
	StartDate    string     `json:"start_date"`
	EndDate      string     `json:"end_date"`
	Observations string     `json:"observations"`
	Status       string     `json:"status"`
	ClosedAt     *time.Time `json:"closed_at,omitempty"`
}

// NewWorkspaceResponse convierte un espacio de dominio.
func NewWorkspaceResponse(w domain.Workspace) WorkspaceResponse {
	return WorkspaceResponse{
		ID:           w.ID,
		Type:         string(w.Type),
		Name:         w.Name,
		PeriodID:     w.PeriodID,
		ProfessorID:  w.ProfessorID,
		StartDate:    domain.FormatDate(w.StartDate),
		EndDate:      domain.FormatDate(w.EndDate),
		Observations: w.Observations,
		Status:       string(w.Status),
		ClosedAt:     w.ClosedAt,
	}
}

// WorkspaceListResponse es una página del listado de cursos y proyectos.
type WorkspaceListResponse struct {
	Items  []WorkspaceResponse `json:"items"`
	Total  int                 `json:"total"`
	Limit  int                 `json:"limit"`
	Offset int                 `json:"offset"`
}

// NewWorkspaceListResponse convierte una página del caso de uso.
func NewWorkspaceListResponse(p workspaces.Page) WorkspaceListResponse {
	items := make([]WorkspaceResponse, len(p.Workspaces))
	for i, w := range p.Workspaces {
		items[i] = NewWorkspaceResponse(w)
	}
	return WorkspaceListResponse{Items: items, Total: p.Total, Limit: p.Limit, Offset: p.Offset}
}
