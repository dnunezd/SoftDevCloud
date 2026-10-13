// Package bootstrap es el composition root: construye y conecta los adaptadores
// concretos con los casos de uso a partir de la configuración.
//
// Es el único paquete que conoce a la vez las implementaciones concretas
// (postgres, asynq, local/gcs, claude/fake, ...) y la capa de aplicación.
package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/inbound/rest"
	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/outbound/clock"
	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/outbound/postgres"
	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/outbound/security"
	"github.com/dnunezd/SoftDevCloud/backend/internal/app/auth"
	"github.com/dnunezd/SoftDevCloud/backend/internal/app/users"
	"github.com/dnunezd/SoftDevCloud/backend/internal/config"
)

// API agrupa las dependencias del proceso api.
type API struct {
	Handler http.Handler
	pool    *pgxpool.Pool
}

// Close libera los recursos del proceso.
func (a *API) Close() {
	a.pool.Close()
}

// NewAPI construye el proceso api: conexión a la base de datos, migraciones,
// casos de uso, administrador inicial y router HTTP.
func NewAPI(ctx context.Context, cfg config.Config, logger *slog.Logger) (*API, error) {
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	app, err := build(ctx, cfg, logger, pool)
	if err != nil {
		pool.Close()
		return nil, err
	}
	return app, nil
}

func build(ctx context.Context, cfg config.Config, logger *slog.Logger, pool *pgxpool.Pool) (*API, error) {
	if cfg.AutoMigrate {
		if err := postgres.Migrate(cfg.DatabaseURL); err != nil {
			return nil, err
		}
		logger.Info("migraciones aplicadas")
	}

	userRepo := postgres.NewUserRepository(pool)
	sessionRepo := postgres.NewSessionRepository(pool)
	hasher := security.NewBcryptHasher(cfg.BcryptCost)
	systemClock := clock.System{}
	tokens := security.NewJWTService(cfg.JWTSecret, systemClock.Now)

	authService := auth.NewService(userRepo, sessionRepo, hasher, tokens, systemClock, cfg.JWTTTL)
	userService := users.NewService(userRepo, hasher)

	if err := ensureAdmin(ctx, cfg.Admin, userService, logger); err != nil {
		return nil, err
	}
	purged, err := authService.PurgeExpiredSessions(ctx)
	if err != nil {
		return nil, err
	}
	if purged > 0 {
		logger.Info("sesiones vencidas eliminadas", slog.Int("cantidad", purged))
	}

	if cfg.Env != "development" {
		gin.SetMode(gin.ReleaseMode)
	}
	router, err := rest.NewRouter(rest.Dependencies{
		Auth:   authService,
		Users:  userService,
		Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("construir router: %w", err)
	}
	return &API{Handler: router, pool: pool}, nil
}

func ensureAdmin(ctx context.Context, admin config.AdminBootstrap, userService *users.Service, logger *slog.Logger) error {
	if admin.Email == "" {
		logger.Warn("BOOTSTRAP_ADMIN_EMAIL no está definido: no se crea administrador inicial")
		return nil
	}
	created, err := userService.EnsureAdmin(ctx, admin.FullName, admin.Email, admin.Password)
	if err != nil {
		return fmt.Errorf("crear administrador inicial: %w", err)
	}
	if created {
		logger.Info("administrador inicial creado", slog.String("email", admin.Email))
	}
	return nil
}
