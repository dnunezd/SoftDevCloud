// Package clock implementa ports.Clock con la hora del sistema. La zona horaria
// configurada (APP_TIMEZONE) se aplicará al calcular la semana operativa.
package clock

import "time"

// System entrega la hora real del sistema.
type System struct{}

// Now devuelve la hora actual.
func (System) Now() time.Time {
	return time.Now()
}
