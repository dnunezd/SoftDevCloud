package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

func graduate(hours int) domain.Assignment {
	return domain.Assignment{Role: domain.RoleGraduateAssistant, WeeklyHours: hours}
}

func monitor(hours int) domain.Assignment {
	return domain.Assignment{Role: domain.RoleMonitor, WeeklyHours: hours}
}

func TestMaxMonitorHoursFor_RoundsUp(t *testing.T) {
	tests := map[int]int{
		22: 9, // 8,8 -> 9 (ejemplo de RN-17)
		12: 5, // 4,8 -> 5 (ejemplo de RN-18)
		10: 4, // 4,0 exacto: no se redondea
		5:  2, // 2,0
		1:  1, // 0,4 -> 1
		0:  0,
	}
	for graduateHours, want := range tests {
		assert.Equal(t, want, domain.MaxMonitorHoursFor(graduateHours), "AG %d h", graduateHours)
	}
}

func TestWorkloadRule_Validate(t *testing.T) {
	tests := []struct {
		name      string
		active    []domain.Assignment
		candidate domain.Assignment
		want      error
	}{
		{"primera vinculación de AG con 22 h", nil, graduate(22), nil},
		{"AG supera 22 h en una sola vinculación", nil, graduate(23), domain.ErrGraduateHoursExceeded},
		{"AG supera 22 h sumando vinculaciones", []domain.Assignment{graduate(12)}, graduate(11), domain.ErrGraduateHoursExceeded},
		{"AG suma exactamente 22 h", []domain.Assignment{graduate(12)}, graduate(10), nil},

		{"tres monitorías de 4 h", []domain.Assignment{monitor(4), monitor(4)}, monitor(4), nil},
		{"cuarta monitoría", []domain.Assignment{monitor(3), monitor(3), monitor(3)}, monitor(1), domain.ErrMonitorCountExceeded},
		{"monitoría suma 12 h", []domain.Assignment{monitor(6)}, monitor(6), nil},
		{"monitoría supera 12 h", []domain.Assignment{monitor(6)}, monitor(7), domain.ErrMonitorHoursExceeded},

		{"AG 22 h permite 9 h de monitoría", []domain.Assignment{graduate(22)}, monitor(9), nil},
		{"AG 22 h no permite 10 h de monitoría", []domain.Assignment{graduate(22)}, monitor(10), domain.ErrMixedHoursExceeded},
		{"AG 12 h permite 5 h de monitoría", []domain.Assignment{graduate(12), monitor(3)}, monitor(2), nil},
		{"AG 12 h rechaza la segunda monitoría de 3 h (RN-18)", []domain.Assignment{graduate(12), monitor(3)}, monitor(3), domain.ErrMixedHoursExceeded},
		{"agregar un AG después de las monitorías", []domain.Assignment{monitor(6)}, graduate(12), domain.ErrMixedHoursExceeded},
		{"sin AG solo aplica el tope de 12 h", []domain.Assignment{monitor(5)}, monitor(7), nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := domain.WorkloadRule{}.Validate(tc.active, tc.candidate)

			if tc.want == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestWorkloadRule_ValidateDoesNotModifyActiveSlice(t *testing.T) {
	active := make([]domain.Assignment, 1, 4)
	active[0] = graduate(10)

	_ = domain.WorkloadRule{}.Validate(active, monitor(2))

	assert.Len(t, active, 1)
	assert.Equal(t, domain.Assignment{}, active[:2][1], "no escribe en la capacidad sobrante del slice")
}
