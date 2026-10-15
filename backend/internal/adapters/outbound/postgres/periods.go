package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

const (
	periodsCodeUniqueKey = "academic_periods_code_key"

	selectPeriod = `
SELECT id::text, code, start_date, end_date, creation_deadline, status
FROM academic_periods
`
	selectPeriodByID = selectPeriod + `WHERE id = $1`
	selectPeriods    = selectPeriod + `ORDER BY start_date DESC, code`
	insertPeriod     = `
INSERT INTO academic_periods (code, start_date, end_date, creation_deadline, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING id::text`
	closePeriod  = `UPDATE academic_periods SET status = 'CERRADO' WHERE id = $1 AND status = 'ACTIVO'`
	periodExists = `SELECT EXISTS (SELECT 1 FROM academic_periods WHERE id = $1)`
)

// PeriodRepository implementa ports.PeriodRepository sobre PostgreSQL.
type PeriodRepository struct {
	pool *pgxpool.Pool
}

// NewPeriodRepository crea el repositorio.
func NewPeriodRepository(pool *pgxpool.Pool) *PeriodRepository {
	return &PeriodRepository{pool: pool}
}

// Create inserta el período. El código es único.
func (r *PeriodRepository) Create(ctx context.Context, period domain.AcademicPeriod) (domain.AcademicPeriod, error) {
	err := r.pool.QueryRow(ctx, insertPeriod,
		period.Code, period.StartDate, period.EndDate, period.CreationDeadline, string(period.Status)).Scan(&period.ID)
	if isUniqueViolation(err, periodsCodeUniqueKey) {
		return domain.AcademicPeriod{}, domain.ErrPeriodCodeExists
	}
	if err != nil {
		return domain.AcademicPeriod{}, fmt.Errorf("insertar período: %w", err)
	}
	return period, nil
}

// GetByID busca un período por su ID. Un ID con formato inválido se trata como inexistente.
func (r *PeriodRepository) GetByID(ctx context.Context, id string) (domain.AcademicPeriod, error) {
	if !isUUID(id) {
		return domain.AcademicPeriod{}, domain.ErrPeriodNotFound
	}
	period, err := scanPeriod(r.pool.QueryRow(ctx, selectPeriodByID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AcademicPeriod{}, domain.ErrPeriodNotFound
	}
	if err != nil {
		return domain.AcademicPeriod{}, fmt.Errorf("consultar período: %w", err)
	}
	return period, nil
}

// List devuelve todos los períodos, del que inicia más tarde al más antiguo.
func (r *PeriodRepository) List(ctx context.Context) ([]domain.AcademicPeriod, error) {
	rows, err := r.pool.Query(ctx, selectPeriods)
	if err != nil {
		return nil, fmt.Errorf("listar períodos: %w", err)
	}
	periods, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.AcademicPeriod, error) {
		return scanPeriod(row)
	})
	if err != nil {
		return nil, fmt.Errorf("leer períodos: %w", err)
	}
	return periods, nil
}

// MarkClosed cierra el período con un UPDATE condicionado al estado ACTIVO, así
// dos cierres simultáneos no se pisan: solo uno cambia la fila.
func (r *PeriodRepository) MarkClosed(ctx context.Context, id string) error {
	if !isUUID(id) {
		return domain.ErrPeriodNotFound
	}
	tag, err := r.pool.Exec(ctx, closePeriod, id)
	if err != nil {
		return fmt.Errorf("cerrar período: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return nil
	}
	var exists bool
	if err := r.pool.QueryRow(ctx, periodExists, id).Scan(&exists); err != nil {
		return fmt.Errorf("consultar período: %w", err)
	}
	if !exists {
		return domain.ErrPeriodNotFound
	}
	return domain.ErrPeriodClosed
}

func scanPeriod(row pgx.Row) (domain.AcademicPeriod, error) {
	var (
		period domain.AcademicPeriod
		status string
	)
	if err := row.Scan(&period.ID, &period.Code, &period.StartDate, &period.EndDate, &period.CreationDeadline, &status); err != nil {
		return domain.AcademicPeriod{}, err
	}
	period.Status = domain.PeriodStatus(status)
	return period, nil
}
