// Command api expone la API REST del sistema (proceso sin estado).
//
// Secuencia de arranque:
//  1. Cargar configuración desde variables de entorno (internal/config).
//  2. Construir dependencias en el composition root (internal/bootstrap), que
//     además aplica las migraciones y crea el administrador inicial.
//  3. Servir HTTP en :$PORT hasta recibir SIGTERM o SIGINT, y apagar de forma ordenada.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/inbound/rest"
	"github.com/dnunezd/SoftDevCloud/backend/internal/bootstrap"
	"github.com/dnunezd/SoftDevCloud/backend/internal/config"
)

func main() {
	if err := run(); err != nil {
		slog.Error("el proceso api terminó con error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuración inválida: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := bootstrap.NewAPI(ctx, cfg, logger)
	if err != nil {
		return err
	}
	defer app.Close()

	return rest.Run(ctx, ":"+cfg.Port, app.Handler, logger)
}
