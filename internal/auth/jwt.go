package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/obrenoalvim/back-template-go/internal/models"
)

type AccessClaims struct {
	Email string      `json:"email"`
	Role  models.Role `json:"role"`
	Type  string      `json:"type"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	Type string `json:"type"`
	JTI  string `json:"jti"`
	jwt.RegisteredClaims
}

func CreateAccessToken(secret string, ttlMinutes int, userID uuid.UUID, email string, role models.Role) (string, error) {
	now := time.Now()
	claims := AccessClaims{
		Email: email,
		Role:  role,
		Type:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(ttlMinutes) * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// CreateRefreshToken returns (token, jti, expiresAt) — jti is persisted server-side so the
// token can be revoked/rotated (deleted on refresh/logout, not just left to expire).
func CreateRefreshToken(secret string, ttlDays int, userID uuid.UUID) (string, string, time.Time, error) {
	now := time.Now()
	jti := uuid.NewString()
	expiresAt := now.Add(time.Duration(ttlDays) * 24 * time.Hour)
	claims := RefreshClaims{
		Type: "refresh",
		JTI:  jti,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	return signed, jti, expiresAt, err
}

func ParseAccessToken(secret, tokenString string) (*AccessClaims, error) {
	claims := &AccessClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims.Type != "access" {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

func ParseRefreshToken(secret, tokenString string) (*RefreshClaims, error) {
	claims := &RefreshClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims.Type != "refresh" {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
