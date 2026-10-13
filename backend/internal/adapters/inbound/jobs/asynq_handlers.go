// Package jobs es el adaptador de entrada asíncrono: registra los manejadores
// de la cola y los conecta con los casos de uso (p. ej. "report:generate" ->
// reports.Generator).
//
// Evolución: en la Entrega 4 se agrega un adaptador de entrada Pub/Sub (push)
// que invoca el mismo caso de uso, y este paquete se retira junto con Asynq.
package jobs
