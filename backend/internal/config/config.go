// Package config carga la configuración desde variables de entorno (12-factor).
//
// Es el único punto que lee el entorno. Los "drivers" (STORAGE_DRIVER,
// QUEUE_DRIVER, LLM_PROVIDER) permitirán cambiar de adaptador entre entregas
// sin modificar el dominio ni los casos de uso. Ver .env.example.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
	_ "time/tzdata" // zonas horarias embebidas: APP_TIMEZONE funciona aunque la imagen no traiga tzdata
)

// Límites aceptados para BCRYPT_COST (los de golang.org/x/crypto/bcrypt).
const (
	minBcryptCost   = 4
	maxBcryptCost   = 31
	minJWTSecretLen = 32
)

// Config es la configuración del proceso.
type Config struct {
	Env  string
	Port string
	// Location es la zona horaria académica (APP_TIMEZONE): define qué día es
	// "hoy" para la ventana de creación de espacios y, desde la Entrega 1, el
	// corte semanal de lunes a domingo.
	Location    *time.Location
	LogLevel    slog.Level
	DatabaseURL string
	AutoMigrate bool
	JWTSecret   string
	JWTTTL      time.Duration
	BcryptCost  int
	Admin       AdminBootstrap
}

// AdminBootstrap son los datos del administrador inicial. Si Email está vacío
// no se crea ninguno.
type AdminBootstrap struct {
	FullName string
	Email    string
	Password string
}

// Load lee y valida la configuración. Reporta todos los problemas a la vez.
func Load() (Config, error) {
	var errs []error
	cfg := Config{
		Env:         getenv("APP_ENV", "development"),
		Port:        getenv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		Admin: AdminBootstrap{
			FullName: getenv("BOOTSTRAP_ADMIN_NAME", "Administrador del sistema"),
			Email:    os.Getenv("BOOTSTRAP_ADMIN_EMAIL"),
			Password: os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
		},
	}

	location, err := time.LoadLocation(getenv("APP_TIMEZONE", "America/Bogota"))
	if err != nil {
		errs = append(errs, fmt.Errorf("APP_TIMEZONE: %w", err))
	}
	cfg.Location = location

	if err := cfg.LogLevel.UnmarshalText([]byte(getenv("LOG_LEVEL", "info"))); err != nil {
		errs = append(errs, fmt.Errorf("LOG_LEVEL: %w", err))
	}

	autoMigrate, err := strconv.ParseBool(getenv("DB_AUTO_MIGRATE", "true"))
	if err != nil {
		errs = append(errs, fmt.Errorf("DB_AUTO_MIGRATE: %w", err))
	}
	cfg.AutoMigrate = autoMigrate

	ttl, err := time.ParseDuration(getenv("JWT_TTL", "8h"))
	if err != nil || ttl <= 0 {
		errs = append(errs, errors.New("JWT_TTL debe ser una duración positiva, p. ej. 8h"))
	}
	cfg.JWTTTL = ttl

	cost, err := strconv.Atoi(getenv("BCRYPT_COST", "10"))
	if err != nil || cost < minBcryptCost || cost > maxBcryptCost {
		errs = append(errs, fmt.Errorf("BCRYPT_COST debe ser un entero entre %d y %d", minBcryptCost, maxBcryptCost))
	}
	cfg.BcryptCost = cost

	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL es obligatoria"))
	}
	if len(cfg.JWTSecret) < minJWTSecretLen {
		errs = append(errs, fmt.Errorf("JWT_SECRET debe tener al menos %d caracteres", minJWTSecretLen))
	}
	if cfg.Admin.Email != "" && cfg.Admin.Password == "" {
		errs = append(errs, errors.New("BOOTSTRAP_ADMIN_PASSWORD es obligatoria si se define BOOTSTRAP_ADMIN_EMAIL"))
	}

	return cfg, errors.Join(errs...)
}

func getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
