package domain

import "time"

// ParseDate interpreta una fecha de calendario en formato AAAA-MM-DD. La fecha
// resultante está en UTC y sin hora, igual que las columnas date de la base de
// datos, y solo se compara como fecha.
func ParseDate(value string) (time.Time, bool) {
	date, err := time.Parse(dateLayout, value)
	return date, err == nil
}

// ParseDateField interpreta una fecha AAAA-MM-DD de un campo de la petición. Si
// no es válida registra el problema en fields y devuelve la fecha cero.
func ParseDateField(fields map[string]string, field, value string) time.Time {
	date, ok := ParseDate(value)
	if !ok {
		fields[field] = "es obligatoria y debe tener formato AAAA-MM-DD"
	}
	return date
}

// FormatDate devuelve la fecha de calendario en formato AAAA-MM-DD.
func FormatDate(t time.Time) string {
	return t.Format(dateLayout)
}

// Today devuelve la fecha de calendario de now en la zona loc, como la
// compara el dominio (por ejemplo, la ventana de creación de espacios).
func Today(now time.Time, loc *time.Location) time.Time {
	return calendarDay(now.In(loc))
}
