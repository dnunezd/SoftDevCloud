package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/dnunezd/SoftDevCloud/backend/internal/app/access"
	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

const principalKey = "principal"

// Authenticator valida un token de acceso y resuelve la identidad con sus roles
// vigentes; lo implementa auth.Service.
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (access.Principal, error)
}

// Authenticate exige un token Bearer válido y deja en el contexto la identidad
// con los roles que el usuario tiene en este momento.
func Authenticate(a Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			abort(c, domain.ErrUnauthenticated)
			return
		}
		principal, err := a.Authenticate(c.Request.Context(), token)
		if err != nil {
			abort(c, err)
			return
		}
		c.Set(principalKey, principal)
		c.Next()
	}
}

// PrincipalFrom devuelve la identidad autenticada de la petición.
func PrincipalFrom(c *gin.Context) (access.Principal, bool) {
	value, ok := c.Get(principalKey)
	if !ok {
		return access.Principal{}, false
	}
	principal, ok := value.(access.Principal)
	return principal, ok
}

func bearerToken(header string) (string, bool) {
	scheme, token, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != ""
}
