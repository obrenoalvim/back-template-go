package admin

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/obrenoalvim/back-template-go/internal/models"
)

type UserSummary struct {
	ID    uuid.UUID   `json:"id"`
	Email string      `json:"email"`
	Role  models.Role `json:"role"`
}

type NoteSummary struct {
	ID         uuid.UUID `json:"id"`
	Title      string    `json:"title"`
	OwnerEmail string    `json:"ownerEmail"`
}

type Handlers struct {
	db *gorm.DB
}

func NewHandlers(db *gorm.DB) *Handlers {
	return &Handlers{db: db}
}

func (h *Handlers) ListUsers(c *fiber.Ctx) error {
	var users []models.User
	if err := h.db.Find(&users).Error; err != nil {
		return err
	}

	summaries := make([]UserSummary, 0, len(users))
	for _, u := range users {
		summaries = append(summaries, UserSummary{ID: u.ID, Email: u.Email, Role: u.Role})
	}
	return c.JSON(summaries)
}

// NotesWithOwners loads every note with its Owner eager-loaded through a
// single SQL join, instead of one query per note. Guarded against regressing
// back into an N+1 by the test in query_count_test.go.
func NotesWithOwners(db *gorm.DB) ([]models.Note, error) {
	var notes []models.Note
	err := db.Joins("Owner").Find(&notes).Error
	return notes, err
}

func (h *Handlers) ListNotes(c *fiber.Ctx) error {
	notes, err := NotesWithOwners(h.db)
	if err != nil {
		return err
	}

	summaries := make([]NoteSummary, 0, len(notes))
	for _, n := range notes {
		summaries = append(summaries, NoteSummary{ID: n.ID, Title: n.Title, OwnerEmail: n.Owner.Email})
	}
	return c.JSON(summaries)
}
