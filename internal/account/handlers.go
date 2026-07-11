package account

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/obrenoalvim/back-template-go/internal/apierror"
	"github.com/obrenoalvim/back-template-go/internal/auth"
	"github.com/obrenoalvim/back-template-go/internal/models"
)

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required"`
	NewPassword     string `json:"newPassword" validate:"required,min=8"`
}

type Handlers struct {
	db *gorm.DB
}

func NewHandlers(db *gorm.DB) *Handlers {
	return &Handlers{db: db}
}

func (h *Handlers) ChangePassword(c *fiber.Ctx) error {
	var req ChangePasswordRequest
	if err := apierror.BindAndValidate(c, &req); err != nil {
		return err
	}

	current := auth.GetCurrentUser(c)
	var user models.User
	if err := h.db.First(&user, "id = ?", current.ID).Error; err != nil {
		return apierror.Unauthorized("Current password is incorrect")
	}
	if !auth.VerifyPassword(req.CurrentPassword, user.PasswordHash) {
		return apierror.Unauthorized("Current password is incorrect")
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	user.PasswordHash = hash
	if err := h.db.Save(&user).Error; err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusOK)
}

func (h *Handlers) DeleteAccount(c *fiber.Ctx) error {
	current := auth.GetCurrentUser(c)
	if err := h.db.Delete(&models.User{}, "id = ?", current.ID).Error; err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusOK)
}
