package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

// RequireRoles permite continuar solo si la identidad tiene alguno de los roles
// (RF-01.2). Debe usarse después de Authenticate. La visibilidad por relación
// (qué datos puede ver cada usuario) se valida en internal/app/access.
func RequireRoles(roles ...domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := PrincipalFrom(c)
		if !ok {
			abort(c, domain.ErrUnauthenticated)
			return
		}
		if !principal.HasAnyRole(roles...) {
			abort(c, domain.ErrForbidden)
			return
		}
		c.Next()
	}
}
