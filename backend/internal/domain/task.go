package domain

// Tarea semanal registrada por un monitor o asistente graduado.
//
// Pertenece a una única vinculación (RN-21). Campos mínimos: título,
// descripción, estado, semana, horas enteras y observaciones (RN-22, RN-24).
// Estados: ABIERTO, EN_DESARROLLO, FINALIZADO, sin estados adicionales
// (RN-27, RN-28).
// Políticas: edición y eliminación solo en semana activa; un reporte tardío
// es inmutable; no se admiten tareas en espacios cerrados (RN-30 a RN-34, RN-09).
