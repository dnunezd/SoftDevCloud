package domain

// Lote de reportes y reporte semanal individual generado con IA.
//
// Un lote (profesor + semana) produce un reporte PDF por persona vinculada
// (RN-41, RN-42, RN-44). Estados del reporte: PENDIENTE, PROCESANDO,
// COMPLETADO, FALLIDO. El contenido se construye solo con tareas,
// descripciones, observaciones y horas (RN-43) y no evalúa desempeño (RN-46).
