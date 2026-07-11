package account

import (
	"github.com/gofiber/fiber/v2"
	"github.com/obrenoalvim/back-template-go/internal/auth"
)

func RegisterRoutes(router fiber.Router, h *Handlers, secret string) {
	g := router.Group("/account", auth.RequireAuth(secret))
	g.Patch("/password", h.ChangePassword)
	g.Delete("/", h.DeleteAccount)
}
