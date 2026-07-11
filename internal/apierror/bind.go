package apierror

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// BindAndValidate parses the JSON body into dst and runs struct-tag validation,
// returning a VALIDATION_ERROR ApiError (matching the family's shape) on failure.
func BindAndValidate(c *fiber.Ctx, dst interface{}) error {
	if err := c.BodyParser(dst); err != nil {
		return Validation([]string{"body: invalid JSON"})
	}
	if err := validate.Struct(dst); err != nil {
		var details []string
		for _, fe := range err.(validator.ValidationErrors) {
			details = append(details, fmt.Sprintf("%s: failed on the '%s' rule", fe.Field(), fe.Tag()))
		}
		return Validation(details)
	}
	return nil
}
