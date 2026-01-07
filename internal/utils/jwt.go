package utils

import (
	"errors"
	"fmt"
	"time"
	"tolelom_api/internal/model"

	"github.com/golang-jwt/jwt/v5"
)

// JWT errors
var (
	ErrInvalidToken         = errors.New("invalid token")
	ErrExpiredToken         = errors.New("token has expired")
	ErrTokenNotYetValid     = errors.New("token not yet valid")
	ErrInvalidSigningMethod = errors.New("unexpected signing method")
	ErrMissingClaims        = errors.New("missing required claims")
)

// Claims represents JWT custom claims
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateJWT creates a new JWT token for the given user
func GenerateJWT(user *model.User, secretKey string) (string, error) {
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "tolelom_api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

// ValidateJWT validates a JWT token and returns the claims if valid
func ValidateJWT(tokenString string, secretKey string) (*Claims, error) {
	// Parse and validate the token
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("%w: %v", ErrInvalidSigningMethod, token.Header["alg"])
			}
			return []byte(secretKey), nil
		},
		jwt.WithLeeway(5*time.Second), // Allow 5 second clock skew
	)

	if err != nil {
		// Detailed error handling
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, ErrExpiredToken
		case errors.Is(err, jwt.ErrTokenNotValidYet):
			return nil, ErrTokenNotYetValid
		case errors.Is(err, jwt.ErrTokenMalformed):
			return nil, fmt.Errorf("%w: malformed token", ErrInvalidToken)
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			return nil, fmt.Errorf("%w: invalid signature", ErrInvalidToken)
		default:
			return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
		}
	}

	// Extract claims
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("%w: failed to parse claims", ErrInvalidToken)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	// Validate required fields
	if claims.UserID == 0 {
		return nil, fmt.Errorf("%w: missing user ID", ErrMissingClaims)
	}
	if claims.Username == "" {
		return nil, fmt.Errorf("%w: missing username", ErrMissingClaims)
	}

	return claims, nil
}
