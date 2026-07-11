package auth

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/obrenoalvim/back-template-go/internal/apierror"
	"github.com/obrenoalvim/back-template-go/internal/config"
	"github.com/obrenoalvim/back-template-go/internal/mail"
	"github.com/obrenoalvim/back-template-go/internal/models"
)

const tokenTTL = time.Hour

type Handlers struct {
	db     *gorm.DB
	cfg    config.Config
	mailer *mail.Mailer
}

func NewHandlers(db *gorm.DB, cfg config.Config, mailer *mail.Mailer) *Handlers {
	return &Handlers{db: db, cfg: cfg, mailer: mailer}
}

func (h *Handlers) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := apierror.BindAndValidate(c, &req); err != nil {
		return err
	}

	var existing models.User
	if err := h.db.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		return apierror.Conflict("Email already registered")
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		return err
	}

	token := uuid.NewString()
	expires := time.Now().Add(tokenTTL)
	user := models.User{
		Email:                      req.Email,
		PasswordHash:               hash,
		VerificationToken:          &token,
		VerificationTokenExpiresAt: &expires,
	}
	if err := h.db.Create(&user).Error; err != nil {
		return err
	}

	h.mailer.Send(req.Email, "Verify your email", "Verification token: "+token)
	return apierror.Empty(c, fiber.StatusCreated)
}

func (h *Handlers) VerifyEmail(c *fiber.Ctx) error {
	token := c.Query("token")

	var user models.User
	if err := h.db.Where("verification_token = ?", token).First(&user).Error; err != nil {
		return apierror.NotFound("Invalid verification token")
	}
	if user.VerificationTokenExpiresAt != nil && user.VerificationTokenExpiresAt.Before(time.Now()) {
		return apierror.Conflict("Verification token expired")
	}

	user.EmailVerified = true
	user.VerificationToken = nil
	user.VerificationTokenExpiresAt = nil
	if err := h.db.Save(&user).Error; err != nil {
		return err
	}
	return apierror.Empty(c, fiber.StatusOK)
}

func (h *Handlers) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := apierror.BindAndValidate(c, &req); err != nil {
		return err
	}

	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return apierror.Unauthorized("Invalid email or password")
	}
	if !VerifyPassword(req.Password, user.PasswordHash) {
		return apierror.Unauthorized("Invalid email or password")
	}
	if !user.EmailVerified {
		return apierror.Unauthorized("Email not verified")
	}

	tokens, err := h.issueTokens(&user)
	if err != nil {
		return err
	}
	return c.JSON(tokens)
}

func (h *Handlers) Refresh(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := apierror.BindAndValidate(c, &req); err != nil {
		return err
	}

	claims, err := ParseRefreshToken(h.cfg.JWTSecret, req.RefreshToken)
	if err != nil {
		return apierror.Unauthorized("Invalid or expired refresh token")
	}

	var stored models.RefreshToken
	if err := h.db.Where("jti = ?", claims.JTI).First(&stored).Error; err != nil {
		return apierror.Unauthorized("Invalid or expired refresh token")
	}
	if stored.ExpiresAt.Before(time.Now()) {
		return apierror.Unauthorized("Invalid or expired refresh token")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return apierror.Unauthorized("Invalid or expired refresh token")
	}
	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		return apierror.Unauthorized("Invalid or expired refresh token")
	}

	if err := h.db.Delete(&stored).Error; err != nil {
		return err
	}

	tokens, err := h.issueTokens(&user)
	if err != nil {
		return err
	}
	return c.JSON(tokens)
}

func (h *Handlers) Logout(c *fiber.Ctx) error {
	var req LogoutRequest
	if err := apierror.BindAndValidate(c, &req); err != nil {
		return err
	}

	claims, err := ParseRefreshToken(h.cfg.JWTSecret, req.RefreshToken)
	if err != nil {
		return apierror.Empty(c, fiber.StatusOK) // idempotent
	}
	h.db.Where("jti = ?", claims.JTI).Delete(&models.RefreshToken{})
	return apierror.Empty(c, fiber.StatusOK)
}

func (h *Handlers) ForgotPassword(c *fiber.Ctx) error {
	var req ForgotPasswordRequest
	if err := apierror.BindAndValidate(c, &req); err != nil {
		return err
	}

	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err == nil {
		token := uuid.NewString()
		expires := time.Now().Add(tokenTTL)
		user.ResetToken = &token
		user.ResetTokenExpiresAt = &expires
		if err := h.db.Save(&user).Error; err != nil {
			return err
		}
		h.mailer.Send(req.Email, "Reset your password", "Reset token: "+token)
	}
	// Always 200 — no user-enumeration leak, same response whether or not the email exists.
	return apierror.Empty(c, fiber.StatusOK)
}

func (h *Handlers) ResetPassword(c *fiber.Ctx) error {
	var req ResetPasswordRequest
	if err := apierror.BindAndValidate(c, &req); err != nil {
		return err
	}

	var user models.User
	if err := h.db.Where("reset_token = ?", req.Token).First(&user).Error; err != nil {
		return apierror.NotFound("Invalid reset token")
	}
	if user.ResetTokenExpiresAt != nil && user.ResetTokenExpiresAt.Before(time.Now()) {
		return apierror.Conflict("Reset token expired")
	}

	hash, err := HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	user.PasswordHash = hash
	user.ResetToken = nil
	user.ResetTokenExpiresAt = nil
	if err := h.db.Save(&user).Error; err != nil {
		return err
	}
	return apierror.Empty(c, fiber.StatusOK)
}

func (h *Handlers) issueTokens(user *models.User) (TokenResponse, error) {
	access, err := CreateAccessToken(h.cfg.JWTSecret, h.cfg.JWTAccessTTLMin, user.ID, user.Email, user.Role)
	if err != nil {
		return TokenResponse{}, err
	}
	refresh, jti, expiresAt, err := CreateRefreshToken(h.cfg.JWTSecret, h.cfg.JWTRefreshTTLDays, user.ID)
	if err != nil {
		return TokenResponse{}, err
	}
	rt := models.RefreshToken{UserID: user.ID, JTI: jti, ExpiresAt: expiresAt}
	if err := h.db.Create(&rt).Error; err != nil {
		return TokenResponse{}, err
	}
	return TokenResponse{AccessToken: access, RefreshToken: refresh}, nil
}
