package ports

import "time"

// Clock entrega la hora actual. Se inyecta para que las reglas temporales
// (vencimiento de sesiones, semana activa, reporte tardío, ventana de creación
// de espacios) sean deterministas en las pruebas.
type Clock interface {
	Now() time.Time
}
