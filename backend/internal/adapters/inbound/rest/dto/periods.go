package dto

import (
	"github.com/dnunezd/SoftDevCloud/backend/internal/app/periods"
	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

// CreatePeriodRequest es el cuerpo de POST /api/v1/admin/periods. Las fechas
// van en formato AAAA-MM-DD.
type CreatePeriodRequest struct {
	Code             string `json:"code"`
	StartDate        string `json:"start_date"`
	EndDate          string `json:"end_date"`
	CreationDeadline string `json:"creation_deadline"`
}

// ToInput convierte la petición en la entrada del caso de uso.
func (r CreatePeriodRequest) ToInput() periods.CreateInput {
	return periods.CreateInput{
		Code: r.Code, StartDate: r.StartDate, EndDate: r.EndDate, CreationDeadline: r.CreationDeadline,
	}
}

// PeriodResponse es la representación pública de un período académico.
type PeriodResponse struct {
	ID               string `json:"id"`
	Code             string `json:"code"`
	StartDate        string `json:"start_date"`
	EndDate          string `json:"end_date"`
	CreationDeadline string `json:"creation_deadline"`
	Status           string `json:"status"`
}

// NewPeriodResponse convierte un período de dominio.
func NewPeriodResponse(p domain.AcademicPeriod) PeriodResponse {
	return PeriodResponse{
		ID:               p.ID,
		Code:             p.Code,
		StartDate:        domain.FormatDate(p.StartDate),
		EndDate:          domain.FormatDate(p.EndDate),
		CreationDeadline: domain.FormatDate(p.CreationDeadline),
		Status:           string(p.Status),
	}
}

// PeriodListResponse es el listado de períodos.
type PeriodListResponse struct {
	Items []PeriodResponse `json:"items"`
}

// NewPeriodListResponse convierte una lista de períodos.
func NewPeriodListResponse(list []domain.AcademicPeriod) PeriodListResponse {
	items := make([]PeriodResponse, len(list))
	for i, p := range list {
		items[i] = NewPeriodResponse(p)
	}
	return PeriodListResponse{Items: items}
}
