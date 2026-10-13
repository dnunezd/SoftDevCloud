package ports

// Publicador de trabajos asíncronos: encola la generación de un reporte por
// persona.
//
// E1 a E3: Asynq + Redis. E4: Cloud Pub/Sub (un mensaje por persona).
