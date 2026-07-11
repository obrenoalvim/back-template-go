package apierror

import "github.com/gofiber/fiber/v2"

type ApiError struct {
	Status  int
	Code    string
	Message string
	Details []string
}

func (e *ApiError) Error() string { return e.Message }

func NotFound(message string) *ApiError {
	return &ApiError{Status: fiber.StatusNotFound, Code: "NOT_FOUND", Message: message}
}

func Unauthorized(message string) *ApiError {
	return &ApiError{Status: fiber.StatusUnauthorized, Code: "UNAUTHORIZED", Message: message}
}

func Forbidden(message string) *ApiError {
	return &ApiError{Status: fiber.StatusForbidden, Code: "FORBIDDEN", Message: message}
}

func Conflict(message string) *ApiError {
	return &ApiError{Status: fiber.StatusConflict, Code: "CONFLICT", Message: message}
}

func Validation(details []string) *ApiError {
	return &ApiError{
		Status:  fiber.StatusBadRequest,
		Code:    "VALIDATION_ERROR",
		Message: "Invalid request body",
		Details: details,
	}
}

type errorBody struct {
	Error struct {
		Code    string   `json:"code"`
		Message string   `json:"message"`
		Details []string `json:"details"`
	} `json:"error"`
}

// Handler is Fiber's global error handler — every error surfaced by a route ends up
// here, producing the same {"error":{code,message,details}} shape used across the
// backend template family (back-template-spring/nest/laravel/fastapi).
func Handler(c *fiber.Ctx, err error) error {
	var apiErr *ApiError
	status := fiber.StatusInternalServerError
	code := "INTERNAL_ERROR"
	message := "Unexpected error"
	details := []string{}

	if fe, ok := err.(*fiber.Error); ok {
		status = fe.Code
		message = fe.Message
		code = "HTTP_ERROR"
	}

	if ae, ok := err.(*ApiError); ok {
		apiErr = ae
	}
	if apiErr != nil {
		status = apiErr.Status
		code = apiErr.Code
		message = apiErr.Message
		details = apiErr.Details
	}

	body := errorBody{}
	body.Error.Code = code
	body.Error.Message = message
	body.Error.Details = details
	return c.Status(status).JSON(body)
}
