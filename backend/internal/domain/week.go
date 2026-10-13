package domain

// Semana operativa (value object), unidad básica de seguimiento (RN-20).
//
// Se identifica por la fecha de su lunes; cierra el domingo (RN-29) según la
// zona horaria configurada (APP_TIMEZONE). La hora actual se recibe de un
// puerto Clock para que las reglas temporales sean testeables.
// Reglas asociadas: semana activa vs. cerrada y reporte tardío (RN-30 a RN-34).
