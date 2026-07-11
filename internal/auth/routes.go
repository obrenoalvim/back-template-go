package auth

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func rateLimited(c *fiber.Ctx) error {
	return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
		"error": fiber.Map{"code": "RATE_LIMITED", "message": "Too many requests", "details": []string{}},
	})
}

func authRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:          5,
		Expiration:   time.Minute,
		LimitReached: rateLimited,
	})
}

func RegisterRoutes(router fiber.Router, h *Handlers) {
	g := router.Group("/auth")
	g.Post("/register", authRateLimiter(), h.Register)
	g.Get("/verify-email", h.VerifyEmail)
	g.Post("/login", authRateLimiter(), h.Login)
	g.Post("/refresh", h.Refresh)
	g.Post("/logout", h.Logout)
	g.Post("/forgot-password", h.ForgotPassword)
	g.Post("/reset-password", h.ResetPassword)
}
