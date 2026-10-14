package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

// bogota es America/Bogota (UTC-5, sin horario de verano). Se usa una zona fija
// para que las pruebas no dependan de la base de datos de zonas del sistema.
var bogota = time.FixedZone("America/Bogota", -5*60*60)

func at(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, bogota)
}

func TestWeekContaining_BoundariesBetweenSundayAndMonday(t *testing.T) {
	// 2026-10-12 es lunes.
	tests := []struct {
		name       string
		instant    time.Time
		wantMonday string
	}{
		{"lunes a las 00:00 abre la semana", at(2026, time.October, 12, 0, 0), "2026-10-12"},
		{"miércoles", at(2026, time.October, 14, 15, 30), "2026-10-12"},
		{"domingo a las 23:59 sigue en la semana", at(2026, time.October, 18, 23, 59), "2026-10-12"},
		{"lunes siguiente a las 00:00 abre otra semana", at(2026, time.October, 19, 0, 0), "2026-10-19"},
		{"cruce de mes", at(2026, time.November, 1, 10, 0), "2026-10-26"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantMonday, domain.WeekContaining(tc.instant, bogota).String())
		})
	}
}

func TestWeekContaining_UsesConfiguredZoneNotUTC(t *testing.T) {
	// Domingo 22:00 en Bogotá es lunes 03:00 en UTC: debe seguir en la semana que cierra.
	sundayNight := at(2026, time.October, 18, 22, 0).UTC()

	assert.Equal(t, "2026-10-12", domain.WeekContaining(sundayNight, bogota).String())
	assert.Equal(t, "2026-10-19", domain.WeekContaining(sundayNight, time.UTC).String())
}

func TestWeek_ActiveClosedAndFuture(t *testing.T) {
	week := domain.WeekContaining(at(2026, time.October, 14, 9, 0), bogota)

	tests := []struct {
		name                   string
		now                    time.Time
		active, closed, future bool
	}{
		{"antes del lunes", at(2026, time.October, 11, 23, 59), false, false, true},
		{"lunes 00:00", at(2026, time.October, 12, 0, 0), true, false, false},
		{"domingo 23:59", at(2026, time.October, 18, 23, 59), true, false, false},
		{"lunes siguiente 00:00", at(2026, time.October, 19, 0, 0), false, true, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.active, week.IsActive(tc.now), "activa")
			assert.Equal(t, tc.closed, week.IsClosed(tc.now), "cerrada")
			assert.Equal(t, tc.future, week.IsFuture(tc.now), "futura")
		})
	}
}

func TestParseWeek(t *testing.T) {
	week, err := domain.ParseWeek("2026-10-12", bogota)
	require.NoError(t, err)
	assert.Equal(t, "2026-10-12", week.String())
	assert.True(t, week.Equal(domain.WeekContaining(at(2026, time.October, 15, 8, 0), bogota)))

	for _, value := range []string{"2026-10-13", "2026-10-18", "12/10/2026", "", "2026-13-01"} {
		_, err := domain.ParseWeek(value, bogota)

		var derr *domain.Error
		require.ErrorAs(t, err, &derr, value)
		assert.Equal(t, domain.KindValidation, derr.Kind, value)
		assert.Contains(t, derr.Fields, "week_start", value)
	}
}
