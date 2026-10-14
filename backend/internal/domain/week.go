package domain

import "time"

// dateLayout es el formato de las fechas (sin hora) que intercambian la API y la BD.
const dateLayout = "2006-01-02"

// Week es la semana operativa, unidad básica de seguimiento (RN-20). Es un
// value object: va de lunes a domingo (RN-29) y se identifica por la fecha de
// su lunes, a las 00:00 en la zona horaria con la que se construyó
// (APP_TIMEZONE). La hora actual siempre se recibe como parámetro, nunca se
// lee del sistema, para que las reglas temporales sean deterministas en las
// pruebas.
type Week struct {
	monday time.Time
}

// WeekContaining devuelve la semana que contiene el instante t según la zona loc.
// Un instante de domingo a las 23:59 pertenece a la semana que cierra ese día.
func WeekContaining(t time.Time, loc *time.Location) Week {
	local := t.In(loc)
	daysSinceMonday := (int(local.Weekday()) + 6) % 7
	return Week{monday: time.Date(local.Year(), local.Month(), local.Day()-daysSinceMonday, 0, 0, 0, 0, loc)}
}

// ParseWeek interpreta la fecha (AAAA-MM-DD) del lunes de una semana. Rechaza
// fechas inválidas y fechas que no son lunes (RN-29).
func ParseWeek(value string, loc *time.Location) (Week, error) {
	date, err := time.ParseInLocation(dateLayout, value, loc)
	if err != nil {
		return Week{}, NewValidationError(map[string]string{"week_start": "no tiene un formato de fecha válido (AAAA-MM-DD)"})
	}
	if date.Weekday() != time.Monday {
		return Week{}, NewValidationError(map[string]string{"week_start": "debe ser un lunes"})
	}
	return Week{monday: date}, nil
}

// Monday devuelve el inicio de la semana: el lunes a las 00:00.
func (w Week) Monday() time.Time { return w.monday }

// End devuelve el instante en que cierra la semana: el lunes siguiente a las
// 00:00, excluyente. Equivale al domingo a las 24:00.
func (w Week) End() time.Time {
	return time.Date(w.monday.Year(), w.monday.Month(), w.monday.Day()+7, 0, 0, 0, 0, w.monday.Location())
}

// IsActive indica si now cae dentro de la semana (RN-30).
func (w Week) IsActive(now time.Time) bool {
	return !now.Before(w.monday) && now.Before(w.End())
}

// IsClosed indica si la semana ya terminó en now (RN-31, RN-32).
func (w Week) IsClosed(now time.Time) bool {
	return !now.Before(w.End())
}

// IsFuture indica si la semana aún no empieza en now.
func (w Week) IsFuture(now time.Time) bool {
	return now.Before(w.monday)
}

// Equal indica si ambas semanas tienen el mismo lunes.
func (w Week) Equal(other Week) bool {
	return w.monday.Equal(other.monday)
}

// IsZero indica si la semana no fue inicializada.
func (w Week) IsZero() bool { return w.monday.IsZero() }

// String devuelve la fecha del lunes en formato AAAA-MM-DD.
func (w Week) String() string { return w.monday.Format(dateLayout) }

// calendarDay descarta la hora y la zona de t y conserva solo su fecha del
// calendario, para comparar fechas sin que influyan las horas.
func calendarDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
