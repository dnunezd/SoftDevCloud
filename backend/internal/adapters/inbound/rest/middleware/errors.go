package middleware

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/inbound/rest/dto"
	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

// requestError representa una petición que no se pudo interpretar (JSON o
// parámetros mal formados). Se responde con 400.
type requestError struct {
	err error
}

func (e *requestError) Error() string { return "petición mal formada: " + e.err.Error() }
func (e *requestError) Unwrap() error { return e.err }

// InvalidRequest envuelve un error de lectura de la petición.
func InvalidRequest(err error) error {
	return &requestError{err: err}
}

// Errors traduce el último error registrado con c.Error a una respuesta JSON
// uniforme. Los errores no previstos se registran en el log y se responden con
// 500 sin exponer detalles internos.
func Errors(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}
		err := c.Errors.Last().Err
		status, body := translate(err)
		if status == http.StatusUnauthorized {
			c.Header("WWW-Authenticate", `Bearer realm="api"`)
		}
		if status >= http.StatusInternalServerError {
			logger.ErrorContext(c.Request.Context(), "error interno",
				slog.String("request_id", c.GetString(requestIDKey)),
				slog.String("error", err.Error()))
		}
		c.JSON(status, body)
	}
}

func translate(err error) (int, dto.ErrorResponse) {
	var derr *domain.Error
	if errors.As(err, &derr) {
		return statusFor(derr.Kind), dto.NewError(derr.Code, derr.Message, derr.Fields)
	}
	var rerr *requestError
	if errors.As(err, &rerr) {
		return http.StatusBadRequest, dto.NewError("invalid_request", "la petición está mal formada", nil)
	}
	return http.StatusInternalServerError, dto.NewError("internal_error", "error interno del servidor", nil)
}

func statusFor(kind domain.ErrorKind) int {
	switch kind {
	case domain.KindValidation:
		return http.StatusUnprocessableEntity
	case domain.KindNotFound:
		return http.StatusNotFound
	case domain.KindConflict:
		return http.StatusConflict
	case domain.KindUnauthorized:
		return http.StatusUnauthorized
	case domain.KindForbidden:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

// abort registra el error y detiene la cadena; Errors escribe la respuesta.
func abort(c *gin.Context, err error) {
	_ = c.Error(err)
	c.Abort()
}
