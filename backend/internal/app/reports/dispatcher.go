// Package reports implementa la generación por lotes de reportes semanales con
// IA (RF-14).
//
// dispatcher.go: caso de uso de despacho. Recibe la solicitud del profesor
// para una semana, registra el lote y un reporte PENDIENTE por persona, y
// publica un trabajo por persona en la cola. Responde de inmediato (202).
package reports
