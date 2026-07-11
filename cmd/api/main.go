package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/obrenoalvim/back-template-go/internal/account"
	"github.com/obrenoalvim/back-template-go/internal/admin"
	"github.com/obrenoalvim/back-template-go/internal/apierror"
	"github.com/obrenoalvim/back-template-go/internal/auth"
	"github.com/obrenoalvim/back-template-go/internal/config"
	"github.com/obrenoalvim/back-template-go/internal/db"
	"github.com/obrenoalvim/back-template-go/internal/mail"
	"github.com/obrenoalvim/back-template-go/internal/notes"
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

	mailer := mail.New(cfg)

	app := fiber.New(fiber.Config{ErrorHandler: apierror.Handler})
	app.Use(requestLogger)

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "UP"})
	})

	auth.RegisterRoutes(app, auth.NewHandlers(gormDB, cfg, mailer))
	account.RegisterRoutes(app, account.NewHandlers(gormDB), cfg.JWTSecret)
	admin.RegisterRoutes(app, admin.NewHandlers(gormDB), cfg.JWTSecret)
	notes.RegisterRoutes(app, notes.NewHandlers(gormDB), cfg.JWTSecret)

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

func requestLogger(c *fiber.Ctx) error {
	start := time.Now()
	err := c.Next()
	slog.Info("request",
		"method", c.Method(),
		"path", c.Path(),
		"status", c.Response().StatusCode(),
		"ms", time.Since(start).Milliseconds(),
	)
	return err
}
