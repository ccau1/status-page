package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid or expired authentication token")
	ErrMissingToken = errors.New("missing authentication token")
)

// SignToken generates a stateless signed JWT containing user claims.
func SignToken(claims UserClaims, secret string, ttl time.Duration) (string, error) {
	if secret == "" {
		return "", errors.New("session secret cannot be empty")
	}

	now := time.Now().UTC()
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(ttl))

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signed, nil
}

// VerifyToken validates a JWT and extracts the user claims.
func VerifyToken(tokenString string, secret string) (*UserClaims, error) {
	if tokenString == "" {
		return nil, ErrMissingToken
	}
	if secret == "" {
		return nil, errors.New("session secret cannot be empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
