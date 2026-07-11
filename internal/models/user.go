package models

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"
)

type User struct {
	ID                         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email                      string    `gorm:"uniqueIndex;not null"`
	PasswordHash               string    `gorm:"not null"`
	Role                       Role      `gorm:"type:varchar(20);not null;default:USER"`
	EmailVerified              bool      `gorm:"not null;default:false"`
	VerificationToken          *string
	VerificationTokenExpiresAt *time.Time
	ResetToken                 *string
	ResetTokenExpiresAt        *time.Time
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}
