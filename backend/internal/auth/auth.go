package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	// Purpose is empty for session tokens and "ticket" for short-lived
	// single-purpose tickets (used to authenticate the SSE stream).
	Purpose string `json:"purpose,omitempty"`
	jwt.RegisteredClaims
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func GenerateToken(secret, userID, email, role string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

const purposeTicket = "ticket"

// GenerateTicket issues a short-lived token that is only valid where a ticket is
// accepted (the SSE stream), so it is safe to put in a URL.
func GenerateTicket(secret, userID, email, role string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID:  userID,
		Email:   email,
		Role:    role,
		Purpose: purposeTicket,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseToken validates a session token. Tickets are rejected.
func ParseToken(secret, tokenStr string) (*Claims, error) {
	claims, err := parse(secret, tokenStr)
	if err != nil || claims.Purpose != "" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// ParseTicket validates a ticket. Session tokens are rejected.
func ParseTicket(secret, tokenStr string) (*Claims, error) {
	claims, err := parse(secret, tokenStr)
	if err != nil || claims.Purpose != purposeTicket {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func parse(secret, tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
