package domain

// Límites de carga horaria (RN-14 a RN-18, RF-05).
const (
	MaxGraduateAssistantHours = 22
	MaxMonitorships           = 3
	MaxMonitorHours           = 12
	// MixedHoursPercent es el porcentaje de las horas de asistencia graduada
	// que pueden dedicarse a monitorías cuando la persona tiene ambos roles.
	MixedHoursPercent = 40
)

// MaxMonitorHoursFor devuelve el máximo de horas de monitoría permitido a una
// persona con graduateHours horas como asistente graduado: el 40 % redondeado
// hacia arriba (RN-17, RN-18). Con 22 h son 9 h; con 12 h, 5 h.
func MaxMonitorHoursFor(graduateHours int) int {
	return (graduateHours*MixedHoursPercent + 99) / 100
}

// WorkloadRule es el servicio de dominio que valida la carga horaria de una
// persona. Las validaciones son bloqueantes (RN-19, RF-05.6).
type WorkloadRule struct{}

// Validate evalúa las vinculaciones activas de una persona más la candidata
// (nueva, o ya rectificada). Devuelve el primer límite incumplido, en este
// orden: horas de asistencia graduada, número de monitorías, horas de
// monitoría y regla mixta.
//
// active no debe incluir la vinculación candidata: al rectificar una
// existente, se excluye del conjunto para no contarla dos veces. Es una
// función pura; la exclusión mutua entre vinculaciones concurrentes de la
// misma persona la garantiza la capa de aplicación con una transacción.
func (WorkloadRule) Validate(active []Assignment, candidate Assignment) error {
	var graduateHours, monitorHours, monitorships int
	for _, a := range append(append([]Assignment(nil), active...), candidate) {
		switch a.Role {
		case RoleGraduateAssistant:
			graduateHours += a.WeeklyHours
		case RoleMonitor:
			monitorHours += a.WeeklyHours
			monitorships++
		}
	}

	switch {
	case graduateHours > MaxGraduateAssistantHours:
		return ErrGraduateHoursExceeded
	case monitorships > MaxMonitorships:
		return ErrMonitorCountExceeded
	case monitorHours > MaxMonitorHours:
		return ErrMonitorHoursExceeded
	case graduateHours > 0 && monitorHours > MaxMonitorHoursFor(graduateHours):
		return ErrMixedHoursExceeded
	default:
		return nil
	}
}
