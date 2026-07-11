package notes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/obrenoalvim/back-template-go/internal/apierror"
	"github.com/obrenoalvim/back-template-go/internal/auth"
	"github.com/obrenoalvim/back-template-go/internal/models"
)

type Handlers struct {
	db *gorm.DB
}

func NewHandlers(db *gorm.DB) *Handlers {
	return &Handlers{db: db}
}

func toResponse(n models.Note) NoteResponse {
	return NoteResponse{ID: n.ID, Title: n.Title, Content: n.Content, CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt}
}

func (h *Handlers) findOwned(c *fiber.Ctx) (*models.Note, error) {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return nil, apierror.NotFound("Note not found")
	}
	current := auth.GetCurrentUser(c)

	var note models.Note
	if err := h.db.Where("id = ? AND owner_id = ?", id, current.ID).First(&note).Error; err != nil {
		return nil, apierror.NotFound("Note not found")
	}
	return &note, nil
}

func (h *Handlers) Create(c *fiber.Ctx) error {
	var req NoteRequest
	if err := apierror.BindAndValidate(c, &req); err != nil {
		return err
	}

	current := auth.GetCurrentUser(c)
	note := models.Note{OwnerID: current.ID, Title: req.Title, Content: req.Content}
	if err := h.db.Create(&note).Error; err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(toResponse(note))
}

func (h *Handlers) List(c *fiber.Ctx) error {
	current := auth.GetCurrentUser(c)
	var notes []models.Note
	if err := h.db.Where("owner_id = ?", current.ID).Find(&notes).Error; err != nil {
		return err
	}

	responses := make([]NoteResponse, 0, len(notes))
	for _, n := range notes {
		responses = append(responses, toResponse(n))
	}
	return c.JSON(responses)
}

func (h *Handlers) Get(c *fiber.Ctx) error {
	note, err := h.findOwned(c)
	if err != nil {
		return err
	}
	return c.JSON(toResponse(*note))
}

func (h *Handlers) Update(c *fiber.Ctx) error {
	note, err := h.findOwned(c)
	if err != nil {
		return err
	}

	var req NoteRequest
	if err := apierror.BindAndValidate(c, &req); err != nil {
		return err
	}

	note.Title = req.Title
	note.Content = req.Content
	if err := h.db.Save(note).Error; err != nil {
		return err
	}
	return c.JSON(toResponse(*note))
}

func (h *Handlers) Delete(c *fiber.Ctx) error {
	note, err := h.findOwned(c)
	if err != nil {
		return err
	}
	if err := h.db.Delete(note).Error; err != nil {
		return err
	}
	return apierror.Empty(c, fiber.StatusNoContent)
}
