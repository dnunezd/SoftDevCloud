package rest

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/inbound/rest/dto"
	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/inbound/rest/handlers"
	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/inbound/rest/middleware"
	"github.com/dnunezd/SoftDevCloud/backend/internal/app/auth"
	"github.com/dnunezd/SoftDevCloud/backend/internal/app/users"
	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

// trustedProxies son las redes privadas desde las que se acepta X-Forwarded-For
// (Nginx en Docker; balanceador de GCP desde E3).
var trustedProxies = []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}

// Dependencies son los casos de uso que expone la API.
type Dependencies struct {
	Auth   *auth.Service
	Users  *users.Service
	Logger *slog.Logger
}

// NewRouter construye la tabla de rutas /api/v1 agrupadas por rol. Ver
// docs/wiki/Entrega-1-Arquitectura.md, sección "API REST".
func NewRouter(d Dependencies) (*gin.Engine, error) {
	r := gin.New()
	if err := r.SetTrustedProxies(trustedProxies); err != nil {
		return nil, err
	}
	r.Use(middleware.RequestLogger(d.Logger), middleware.Recovery(d.Logger), middleware.Errors(d.Logger))

	r.GET("/healthz", handlers.Health)

	authHandlers := handlers.NewAuth(d.Auth, d.Users)
	userHandlers := handlers.NewUsers(d.Users)
	sessionHandlers := handlers.NewSessions(d.Auth)

	v1 := r.Group("/api/v1")
	v1.POST("/auth/login", authHandlers.Login)

	authenticated := v1.Group("", middleware.Authenticate(d.Auth))
	authenticated.POST("/auth/logout", authHandlers.Logout)
	authenticated.GET("/me", authHandlers.Me)

	admin := authenticated.Group("/admin", middleware.RequireRoles(domain.RoleAdmin))
	admin.POST("/users", userHandlers.Create)
	admin.GET("/users", userHandlers.List)
	admin.GET("/users/:id", userHandlers.Get)
	admin.PUT("/users/:id/roles", userHandlers.SetRoles)
	admin.GET("/users/:id/sessions", sessionHandlers.List)
	admin.DELETE("/users/:id/sessions", sessionHandlers.RevokeAll)

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, dto.NewError("route_not_found", "la ruta no existe", nil))
	})
	return r, nil
}
