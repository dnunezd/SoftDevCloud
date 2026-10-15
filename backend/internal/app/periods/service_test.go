package periods_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dnunezd/SoftDevCloud/backend/internal/app/periods"
	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
	"github.com/dnunezd/SoftDevCloud/backend/internal/mocks"
)

const periodID = "4a1b2c3d-5e6f-4a7b-8c9d-0e1f2a3b4c5d"

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func validInput() periods.CreateInput {
	return periods.CreateInput{Code: " 2026-10 ", StartDate: "2026-02-01", EndDate: "2026-06-15", CreationDeadline: "2026-02-28"}
}

// RF-03.2: el período registra identificación, fechas y estado.
func TestCreate_RegistersAnActivePeriod(t *testing.T) {
	repo := mocks.NewPeriodRepository(t)
	repo.EXPECT().Create(mock.Anything, domain.AcademicPeriod{
		Code:             "2026-10",
		StartDate:        date(2026, time.February, 1),
		EndDate:          date(2026, time.June, 15),
		CreationDeadline: date(2026, time.February, 28),
		Status:           domain.PeriodActive,
	}).RunAndReturn(func(_ context.Context, p domain.AcademicPeriod) (domain.AcademicPeriod, error) {
		p.ID = periodID
		return p, nil
	})

	created, err := periods.NewService(repo).Create(context.Background(), validInput())

	require.NoError(t, err)
	assert.Equal(t, periodID, created.ID)
	assert.Equal(t, domain.PeriodActive, created.Status, "un período nace activo")
}

func TestCreate_RejectsInvalidData(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*periods.CreateInput)
		fields []string
	}{
		{"código vacío", func(in *periods.CreateInput) { in.Code = "  " }, []string{"code"}},
		{"fecha con formato inválido", func(in *periods.CreateInput) { in.StartDate = "1 de febrero" }, []string{"start_date"}},
		{"fecha ausente", func(in *periods.CreateInput) { in.EndDate = "" }, []string{"end_date"}},
		{"varias fechas y código inválidos a la vez", func(in *periods.CreateInput) {
			in.Code, in.StartDate, in.CreationDeadline = "", "x", "y"
		}, []string{"code", "start_date", "creation_deadline"}},
		{"fecha límite fuera del período", func(in *periods.CreateInput) { in.CreationDeadline = "2026-07-01" }, []string{"creation_deadline"}},
		{"cierre antes del inicio", func(in *periods.CreateInput) { in.EndDate = "2026-01-01" }, []string{"end_date"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := validInput()
			tc.mutate(&in)

			// El repositorio simulado no espera llamadas: una validación fallida no persiste nada.
			_, err := periods.NewService(mocks.NewPeriodRepository(t)).Create(context.Background(), in)

			var derr *domain.Error
			require.ErrorAs(t, err, &derr)
			assert.Equal(t, domain.KindValidation, derr.Kind)
			for _, field := range tc.fields {
				assert.Contains(t, derr.Fields, field)
			}
		})
	}
}

func TestCreate_DuplicateCode(t *testing.T) {
	repo := mocks.NewPeriodRepository(t)
	repo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.AcademicPeriod{}, domain.ErrPeriodCodeExists)

	_, err := periods.NewService(repo).Create(context.Background(), validInput())

	assert.ErrorIs(t, err, domain.ErrPeriodCodeExists)
}

func TestClose(t *testing.T) {
	active := domain.AcademicPeriod{ID: periodID, Code: "2026-10", Status: domain.PeriodActive}

	t.Run("cierra un período activo", func(t *testing.T) {
		repo := mocks.NewPeriodRepository(t)
		repo.EXPECT().GetByID(mock.Anything, periodID).Return(active, nil)
		repo.EXPECT().MarkClosed(mock.Anything, periodID).Return(nil)

		closed, err := periods.NewService(repo).Close(context.Background(), periodID)

		require.NoError(t, err)
		assert.Equal(t, domain.PeriodClosed, closed.Status)
	})

	t.Run("un período cerrado no se cierra otra vez", func(t *testing.T) {
		repo := mocks.NewPeriodRepository(t)
		repo.EXPECT().GetByID(mock.Anything, periodID).Return(domain.AcademicPeriod{ID: periodID, Status: domain.PeriodClosed}, nil)

		_, err := periods.NewService(repo).Close(context.Background(), periodID)

		assert.ErrorIs(t, err, domain.ErrPeriodClosed)
	})

	t.Run("período inexistente", func(t *testing.T) {
		repo := mocks.NewPeriodRepository(t)
		repo.EXPECT().GetByID(mock.Anything, "no-existe").Return(domain.AcademicPeriod{}, domain.ErrPeriodNotFound)

		_, err := periods.NewService(repo).Close(context.Background(), "no-existe")

		assert.ErrorIs(t, err, domain.ErrPeriodNotFound)
	})

	t.Run("otro cierre se adelantó entre la lectura y la escritura", func(t *testing.T) {
		repo := mocks.NewPeriodRepository(t)
		repo.EXPECT().GetByID(mock.Anything, periodID).Return(active, nil)
		repo.EXPECT().MarkClosed(mock.Anything, periodID).Return(domain.ErrPeriodClosed)

		_, err := periods.NewService(repo).Close(context.Background(), periodID)

		assert.ErrorIs(t, err, domain.ErrPeriodClosed)
	})
}

func TestListAndGet(t *testing.T) {
	repo := mocks.NewPeriodRepository(t)
	want := []domain.AcademicPeriod{{ID: periodID, Code: "2026-20"}, {ID: "otro", Code: "2026-10"}}
	repo.EXPECT().List(mock.Anything).Return(want, nil)
	repo.EXPECT().GetByID(mock.Anything, periodID).Return(want[0], nil)
	service := periods.NewService(repo)

	list, err := service.List(context.Background())
	require.NoError(t, err)
	assert.Equal(t, want, list)

	one, err := service.Get(context.Background(), periodID)
	require.NoError(t, err)
	assert.Equal(t, want[0], one)
}
