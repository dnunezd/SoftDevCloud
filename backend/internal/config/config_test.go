package config_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dnunezd/SoftDevCloud/backend/internal/config"
)

func setValidEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@db:5432/app")
	t.Setenv("JWT_SECRET", "clave-de-pruebas-de-al-menos-32-caracteres")
	t.Setenv("BOOTSTRAP_ADMIN_EMAIL", "")
	t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "")
	for _, key := range []string{"APP_ENV", "PORT", "LOG_LEVEL", "DB_AUTO_MIGRATE", "JWT_TTL", "BCRYPT_COST"} {
		t.Setenv(key, "")
	}
}

func TestLoad_AppliesDefaults(t *testing.T) {
	setValidEnv(t)

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, slog.LevelInfo, cfg.LogLevel)
	assert.True(t, cfg.AutoMigrate)
	assert.Equal(t, 8*time.Hour, cfg.JWTTTL)
	assert.Equal(t, 10, cfg.BcryptCost)
}

func TestLoad_ReadsOverrides(t *testing.T) {
	setValidEnv(t)
	t.Setenv("PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("JWT_TTL", "30m")
	t.Setenv("BCRYPT_COST", "12")
	t.Setenv("BOOTSTRAP_ADMIN_EMAIL", "admin@uni.edu")
	t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "secreta123")

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, slog.LevelDebug, cfg.LogLevel)
	assert.Equal(t, 30*time.Minute, cfg.JWTTTL)
	assert.Equal(t, 12, cfg.BcryptCost)
	assert.Equal(t, "admin@uni.edu", cfg.Admin.Email)
}

func TestLoad_ReportsEveryProblem(t *testing.T) {
	setValidEnv(t)
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "corta")
	t.Setenv("JWT_TTL", "nunca")
	t.Setenv("BCRYPT_COST", "50")
	t.Setenv("BOOTSTRAP_ADMIN_EMAIL", "admin@uni.edu")

	_, err := config.Load()

	require.Error(t, err)
	for _, key := range []string{"DATABASE_URL", "JWT_SECRET", "JWT_TTL", "BCRYPT_COST", "BOOTSTRAP_ADMIN_PASSWORD"} {
		assert.Contains(t, err.Error(), key)
	}
}
