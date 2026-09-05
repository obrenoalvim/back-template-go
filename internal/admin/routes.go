package admin

import (
	"github.com/gofiber/fiber/v2"
	"github.com/obrenoalvim/back-template-go/internal/auth"
)

func RegisterRoutes(router fiber.Router, h *Handlers, secret string) {
	g := router.Group("/admin", auth.RequireAuth(secret), auth.RequireAdmin)
	g.Get("/users", h.ListUsers)
	g.Get("/notes", h.ListNotes)
}
