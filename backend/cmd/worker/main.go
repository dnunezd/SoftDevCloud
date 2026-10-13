// Command worker consume trabajos asíncronos (generación de reportes PDF con IA).
//
// Secuencia de arranque prevista:
//  1. Cargar configuración (internal/config).
//  2. Construir dependencias (internal/bootstrap).
//  3. Registrar los manejadores de internal/adapters/inbound/jobs.
//  4. Consumir la cola con WORKER_CONCURRENCY y apagado ordenado ante SIGTERM.
//
// Evolución: en la Entrega 4 este proceso se retira al migrar a Pub/Sub push
// (el consumo pasa a un endpoint HTTP del servicio en Cloud Run).
package main

func main() {}
