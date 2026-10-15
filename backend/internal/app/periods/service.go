// Package periods implementa la gestión de períodos académicos: creación,
// consulta y cierre (RF-03.2, RN-07).
//
// Solo el administrador crea y cierra períodos; los profesores los consultan
// para elegir el período de sus cursos y proyectos. Esa restricción de rol la
// aplica el router.
package periods

import (
	"context"
	"fmt"
	"strings"

	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
	"github.com/dnunezd/SoftDevCloud/backend/internal/ports"
)

// Service gestiona los períodos académicos.
type Service struct {
	repo ports.PeriodRepository
}

// NewService crea el servicio de períodos.
func NewService(repo ports.PeriodRepository) *Service {
	return &Service{repo: repo}
}

// CreateInput son los datos para crear un período. Las fechas llegan como
// texto AAAA-MM-DD.
type CreateInput struct {
	Code             string
	StartDate        string
	EndDate          string
	CreationDeadline string
}

// Create registra un período ACTIVO con su ventana de creación de espacios
// (RF-03.2). Un período no se crea ya cerrado.
func (s *Service) Create(ctx context.Context, in CreateInput) (domain.AcademicPeriod, error) {
	fields := map[string]string{}
	period := domain.AcademicPeriod{Code: strings.TrimSpace(in.Code), Status: domain.PeriodActive}
	period.StartDate = domain.ParseDateField(fields, "start_date", in.StartDate)
	period.EndDate = domain.ParseDateField(fields, "end_date", in.EndDate)
	period.CreationDeadline = domain.ParseDateField(fields, "creation_deadline", in.CreationDeadline)

	if len(fields) > 0 {
		// Con fechas inválidas no se puede validar la coherencia entre ellas,
		// pero sí se informa de un código vacío en la misma respuesta.
		if period.Code == "" {
			fields["code"] = "es obligatorio"
		}
		return domain.AcademicPeriod{}, domain.NewValidationError(fields)
	}
	if err := period.Validate(); err != nil {
		return domain.AcademicPeriod{}, err
	}

	created, err := s.repo.Create(ctx, period)
	if err != nil {
		return domain.AcademicPeriod{}, fmt.Errorf("crear período: %w", err)
	}
	return created, nil
}

// List devuelve todos los períodos, del más reciente al más antiguo.
func (s *Service) List(ctx context.Context) ([]domain.AcademicPeriod, error) {
	periods, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listar períodos: %w", err)
	}
	return periods, nil
}

// Get consulta un período por su ID.
func (s *Service) Get(ctx context.Context, id string) (domain.AcademicPeriod, error) {
	period, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.AcademicPeriod{}, fmt.Errorf("consultar período: %w", err)
	}
	return period, nil
}

// Close cierra un período: deja de admitir la creación de cursos y proyectos.
// Los espacios ya creados no cambian (RF-03.2).
func (s *Service) Close(ctx context.Context, id string) (domain.AcademicPeriod, error) {
	period, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.AcademicPeriod{}, fmt.Errorf("consultar período: %w", err)
	}
	if err := period.Close(); err != nil {
		return domain.AcademicPeriod{}, err
	}
	if err := s.repo.MarkClosed(ctx, id); err != nil {
		return domain.AcademicPeriod{}, fmt.Errorf("cerrar período: %w", err)
	}
	return period, nil
}
