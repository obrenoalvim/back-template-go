package apierror

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New()
	// Report the JSON field name (e.g. "email") instead of the Go struct field name
	// ("Email"), matching the lowercase-field error details used across the rest of
	// the backend template family (Spring/Nest/Laravel/FastAPI).
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return field.Name
		}
		return name
	})
	return v
}

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
