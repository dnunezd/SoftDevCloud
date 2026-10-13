package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// connectAttempts y connectBackoff dan margen a que la base de datos termine de
// arrancar (contenedor, Cloud SQL) antes de fallar.
const (
	connectAttempts = 10
	connectBackoff  = 2 * time.Second
)

// NewPool crea el pool de conexiones desde DATABASE_URL y verifica la conectividad.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_URL inválida: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("crear pool: %w", err)
	}

	for attempt := 1; ; attempt++ {
		err = pool.Ping(ctx)
		if err == nil {
			return pool, nil
		}
		if attempt == connectAttempts {
			pool.Close()
			return nil, fmt.Errorf("conectar a PostgreSQL tras %d intentos: %w", attempt, err)
		}
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, ctx.Err()
		case <-time.After(connectBackoff):
		}
	}
}
