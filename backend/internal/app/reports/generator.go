package reports

// generator.go: caso de uso de generación, invocado por el consumidor de la
// cola. Recupera los datos semanales de la persona, solicita la síntesis a la
// IA, genera el PDF, lo guarda en FileStorage y marca el reporte COMPLETADO.
//
// Debe ser idempotente: si el reporte ya está COMPLETADO, un reintento o un
// mensaje duplicado no lo regenera.
