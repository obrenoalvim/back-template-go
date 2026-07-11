package server

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/obrenoalvim/back-template-go/internal/account"
	"github.com/obrenoalvim/back-template-go/internal/admin"
	"github.com/obrenoalvim/back-template-go/internal/apierror"
	"github.com/obrenoalvim/back-template-go/internal/auth"
	"github.com/obrenoalvim/back-template-go/internal/config"
	"github.com/obrenoalvim/back-template-go/internal/mail"
	"github.com/obrenoalvim/back-template-go/internal/notes"
)

// New builds the fully-wired Fiber app — shared by cmd/api/main.go and the
// integration tests, so route wiring only lives in one place.
func New(cfg config.Config, gormDB *gorm.DB) *fiber.App {
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

	return app
}

// requestLogger runs the configured ErrorHandler itself (instead of letting Fiber's
// outer dispatcher do it after this middleware returns) so c.Response().StatusCode()
// reflects the real outcome — otherwise every erroring request logs whatever default
// status was set before the handler ran (200), not the 4xx/5xx actually sent.
func requestLogger(c *fiber.Ctx) error {
	start := time.Now()
	err := c.Next()
	if err != nil {
		err = apierror.Handler(c, err)
	}
	slog.Info("request",
		"method", c.Method(),
		"path", c.Path(),
		"status", c.Response().StatusCode(),
		"ms", time.Since(start).Milliseconds(),
	)
	return err
}
