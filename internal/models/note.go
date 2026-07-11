package models

import (
	"time"

	"github.com/google/uuid"
)

type Note struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerID   uuid.UUID `gorm:"type:uuid;not null;index"`
	Owner     User      `gorm:"foreignKey:OwnerID;constraint:OnDelete:CASCADE"`
	Title     string    `gorm:"type:varchar(255);not null"`
	Content   string    `gorm:"type:text;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
