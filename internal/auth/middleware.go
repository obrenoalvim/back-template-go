package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/obrenoalvim/back-template-go/internal/apierror"
	"github.com/obrenoalvim/back-template-go/internal/models"
)

type CurrentUser struct {
	ID    uuid.UUID
	Email string
	Role  models.Role
}

const localsKeyUser = "currentUser"

// RequireAuth is the single dependency every protected route uses — decodes and
// validates the bearer token, no per-route duplication.
func RequireAuth(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			return apierror.Unauthorized("Authentication required")
		}
		tokenString := strings.TrimPrefix(header, "Bearer ")

		claims, err := ParseAccessToken(secret, tokenString)
		if err != nil {
			return apierror.Unauthorized("Authentication required")
		}

		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			return apierror.Unauthorized("Authentication required")
		}

		c.Locals(localsKeyUser, CurrentUser{ID: userID, Email: claims.Email, Role: claims.Role})
		return c.Next()
	}
}

func RequireAdmin(c *fiber.Ctx) error {
	user := GetCurrentUser(c)
	if user.Role != models.RoleAdmin {
		return apierror.Forbidden("Insufficient role")
	}
	return c.Next()
}

func GetCurrentUser(c *fiber.Ctx) CurrentUser {
	return c.Locals(localsKeyUser).(CurrentUser)
}
