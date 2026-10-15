package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

func TestParseDate(t *testing.T) {
	parsed, ok := domain.ParseDate("2026-02-28")
	require.True(t, ok)
	assert.Equal(t, date(2026, time.February, 28), parsed)
	assert.Equal(t, "2026-02-28", domain.FormatDate(parsed))

	for _, invalid := range []string{"", "2026-13-01", "2026-02-30", "28/02/2026", "2026-2-8x"} {
		_, ok := domain.ParseDate(invalid)
		assert.False(t, ok, invalid)
	}
}

func TestParseDateField(t *testing.T) {
	fields := map[string]string{}

	assert.Equal(t, date(2026, time.March, 1), domain.ParseDateField(fields, "start_date", "2026-03-01"))
	assert.Empty(t, fields)

	assert.True(t, domain.ParseDateField(fields, "end_date", "mañana").IsZero())
	assert.Contains(t, fields, "end_date")
}

// El día se decide en la zona académica, no en UTC: a las 23:30 del 31 de agosto
// en Bogotá ya es 1 de septiembre en UTC, pero la ventana sigue abierta.
func TestToday_UsesTheAcademicTimeZone(t *testing.T) {
	bogota, err := time.LoadLocation("America/Bogota")
	require.NoError(t, err)
	instant := time.Date(2026, time.September, 1, 4, 30, 0, 0, time.UTC) // 23:30 del 31 de agosto en Bogotá

	assert.Equal(t, date(2026, time.August, 31), domain.Today(instant, bogota))
	assert.Equal(t, date(2026, time.September, 1), domain.Today(instant, time.UTC))
}
