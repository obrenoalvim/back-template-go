package main

import (
	"log/slog"
	"os"

	"github.com/obrenoalvim/back-template-go/internal/config"
	"github.com/obrenoalvim/back-template-go/internal/db"
	"github.com/obrenoalvim/back-template-go/internal/server"
)

func main() {
	cfg := config.Load()
	configureLogging(cfg)

	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	gormDB, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}

	app := server.New(cfg, gormDB)

	slog.Info("server starting", "port", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func configureLogging(cfg config.Config) {
	level := slog.LevelInfo
	if cfg.LogLevel == "debug" {
		level = slog.LevelDebug
	}

	var handler slog.Handler
	if cfg.Environment == "dev" {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}
	slog.SetDefault(slog.New(handler))
}
