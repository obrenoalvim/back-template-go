package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/obrenoalvim/back-template-go/internal/config"
	"github.com/obrenoalvim/back-template-go/internal/db"
	"github.com/obrenoalvim/back-template-go/internal/models"
	"github.com/obrenoalvim/back-template-go/internal/server"
)

func testConfig() config.Config {
	cfg := config.Load()
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		cfg.DatabaseURL = url
	}
	return cfg
}

func setup(t *testing.T) (*testApp, *gorm.DB) {
	t.Helper()
	cfg := testConfig()

	require.NoError(t, db.Migrate(cfg.DatabaseURL))
	gormDB, err := db.Connect(cfg.DatabaseURL)
	require.NoError(t, err)

	return &testApp{app: server.New(cfg, gormDB)}, gormDB
}

type testApp struct{ app *fiber.App }

func (f *testApp) do(t *testing.T, method, path string, body any, token string) (*http.Response, map[string]any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := f.app.Test(req, 5000)
	require.NoError(t, err)

	respBody, _ := io.ReadAll(resp.Body)
	var parsed map[string]any
	if len(respBody) > 0 {
		_ = json.Unmarshal(respBody, &parsed)
	}
	return resp, parsed
}

func TestFullAuthAndNotesFlow(t *testing.T) {
	app, gormDB := setup(t)
	email := fmt.Sprintf("gotest-%s@example.com", uuid.NewString())
	password := "password123"

	resp, _ := app.do(t, http.MethodPost, "/auth/register", map[string]string{"email": email, "password": password}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp, dup := app.do(t, http.MethodPost, "/auth/register", map[string]string{"email": email, "password": password}, "")
	require.Equal(t, http.StatusConflict, resp.StatusCode)
	require.Equal(t, "CONFLICT", dup["error"].(map[string]any)["code"])

	resp, _ = app.do(t, http.MethodPost, "/auth/login", map[string]string{"email": email, "password": password}, "")
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	var user models.User
	require.NoError(t, gormDB.Where("email = ?", email).First(&user).Error)
	require.NotNil(t, user.VerificationToken)

	resp, _ = app.do(t, http.MethodGet, "/auth/verify-email?token="+*user.VerificationToken, nil, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	resp, login := app.do(t, http.MethodPost, "/auth/login", map[string]string{"email": email, "password": password}, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	accessToken := login["accessToken"].(string)
	refreshToken := login["refreshToken"].(string)

	resp, note := app.do(t, http.MethodPost, "/api/notes/", map[string]string{"title": "Hello", "content": "World"}, accessToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	noteID := note["id"].(string)

	resp, _ = app.do(t, http.MethodGet, "/api/notes/", nil, accessToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	resp, updated := app.do(t, http.MethodPut, "/api/notes/"+noteID, map[string]string{"title": "Updated", "content": "World"}, accessToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "Updated", updated["title"])

	resp, _ = app.do(t, http.MethodDelete, "/api/notes/"+noteID, nil, accessToken)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	resp, _ = app.do(t, http.MethodGet, "/admin/users", nil, accessToken)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)

	resp, refreshed := app.do(t, http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": refreshToken}, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotEmpty(t, refreshed["accessToken"])

	resp, _ = app.do(t, http.MethodDelete, "/account/", nil, accessToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestLoginWithWrongPasswordIsUnauthorized(t *testing.T) {
	app, _ := setup(t)
	email := fmt.Sprintf("gotest-%s@example.com", uuid.NewString())

	resp, _ := app.do(t, http.MethodPost, "/auth/register", map[string]string{"email": email, "password": "password123"}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp, body := app.do(t, http.MethodPost, "/auth/login", map[string]string{"email": email, "password": "wrong-password"}, "")
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	require.Equal(t, "UNAUTHORIZED", body["error"].(map[string]any)["code"])
}
