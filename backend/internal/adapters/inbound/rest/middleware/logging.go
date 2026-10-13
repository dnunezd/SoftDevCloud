package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/inbound/rest/dto"
)

// RequestIDHeader propaga el identificador de petición entre servicios.
const RequestIDHeader = "X-Request-ID"

const (
	requestIDKey       = "request_id"
	maxRequestIDLength = 64
)

// RequestLogger asigna un identificador a cada petición y registra un log de
// acceso estructurado con ruta, estado y latencia. Esta información es la base
// del análisis de las pruebas de carga.
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		id := c.GetHeader(RequestIDHeader)
		if !validRequestID(id) {
			id = newRequestID()
		}
		c.Set(requestIDKey, id)
		c.Header(RequestIDHeader, id)

		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "sin_ruta"
		}
		logger.LogAttrs(c.Request.Context(), slog.LevelInfo, "petición atendida",
			slog.String("request_id", id),
			slog.String("method", c.Request.Method),
			slog.String("route", route),
			slog.Int("status", c.Writer.Status()),
			slog.Float64("latency_ms", float64(time.Since(start).Microseconds())/1000),
			slog.String("client_ip", c.ClientIP()),
		)
	}
}

// Recovery convierte un panic en una respuesta 500 con el formato de error de la API.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.ErrorContext(c.Request.Context(), "panic recuperado",
			slog.String("request_id", c.GetString(requestIDKey)),
			slog.Any("panic", recovered))
		c.AbortWithStatusJSON(http.StatusInternalServerError,
			dto.NewError("internal_error", "error interno del servidor", nil))
	})
}

// validRequestID acepta solo identificadores cortos y con caracteres seguros,
// para no permitir inyección en los logs.
func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLength {
		return false
	}
	for _, c := range id {
		isAlnum := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		if !isAlnum && c != '-' && c != '_' {
			return false
		}
	}
	return true
}

func newRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
