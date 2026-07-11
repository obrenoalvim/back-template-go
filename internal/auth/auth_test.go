package auth

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obrenoalvim/back-template-go/internal/models"
)

func TestPasswordHashRoundtrip(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	require.NoError(t, err)
	assert.NotEqual(t, "correct-horse-battery-staple", hash)
	assert.True(t, VerifyPassword("correct-horse-battery-staple", hash))
	assert.False(t, VerifyPassword("wrong-password", hash))
}

func TestAccessTokenRoundtrip(t *testing.T) {
	userID := uuid.New()
	token, err := CreateAccessToken("test-secret", 15, userID, "a@b.com", models.RoleUser)
	require.NoError(t, err)

	claims, err := ParseAccessToken("test-secret", token)
	require.NoError(t, err)
	assert.Equal(t, userID.String(), claims.Subject)
	assert.Equal(t, "a@b.com", claims.Email)
	assert.Equal(t, models.RoleUser, claims.Role)
	assert.Equal(t, "access", claims.Type)
}

func TestRefreshTokenRoundtrip(t *testing.T) {
	userID := uuid.New()
	token, jti, _, err := CreateRefreshToken("test-secret", 30, userID)
	require.NoError(t, err)

	claims, err := ParseRefreshToken("test-secret", token)
	require.NoError(t, err)
	assert.Equal(t, userID.String(), claims.Subject)
	assert.Equal(t, jti, claims.JTI)
	assert.Equal(t, "refresh", claims.Type)
}
