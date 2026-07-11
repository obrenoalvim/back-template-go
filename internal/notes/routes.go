package notes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/obrenoalvim/back-template-go/internal/auth"
)

func RegisterRoutes(router fiber.Router, h *Handlers, secret string) {
	g := router.Group("/api/notes", auth.RequireAuth(secret))
	g.Post("/", h.Create)
	g.Get("/", h.List)
	g.Get("/:id", h.Get)
	g.Put("/:id", h.Update)
	g.Delete("/:id", h.Delete)
}
