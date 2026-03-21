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
	ErrInvalidTokenType     = errors.New("invalid token type")
)

// Claims represents JWT custom claims
type Claims struct {
	UserID       uint   `json:"user_id"`
	Username     string `json:"username"`
	TokenType    string `json:"token_type"`                   // "access" or "refresh"
	TokenVersion int    `json:"token_version,omitempty"` // refresh token에만 사용
	jwt.RegisteredClaims
}

// GenerateTokenPair creates a new access token (15min) and refresh token (7 days) for the given user
func GenerateTokenPair(user *model.User, secretKey string) (accessToken string, refreshToken string, err error) {
	now := time.Now()

	// Access token: 15 minutes
	accessClaims := Claims{
		UserID:    user.ID,
		Username:  user.Username,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "tolelom_api",
		},
	}
	accessToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(secretKey))
	if err != nil {
		return "", "", err
	}

	// Refresh token: 7 days
	refreshClaims := Claims{
		UserID:       user.ID,
		Username:     user.Username,
		TokenType:    "refresh",
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "tolelom_api",
		},
	}
	refreshToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(secretKey))
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// validateToken parses and validates a JWT token, returning the claims
func validateToken(tokenString string, secretKey string) (*Claims, error) {
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

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("%w: failed to parse claims", ErrInvalidToken)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.UserID == 0 {
		return nil, fmt.Errorf("%w: missing user ID", ErrMissingClaims)
	}
	if claims.Username == "" {
		return nil, fmt.Errorf("%w: missing username", ErrMissingClaims)
	}

	return claims, nil
}

// ValidateAccessToken validates a JWT token and only accepts access tokens
func ValidateAccessToken(tokenString string, secretKey string) (*Claims, error) {
	claims, err := validateToken(tokenString, secretKey)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != "access" {
		return nil, fmt.Errorf("%w: expected access token", ErrInvalidTokenType)
	}
	return claims, nil
}

// ValidateRefreshToken validates a JWT token and only accepts refresh tokens
func ValidateRefreshToken(tokenString string, secretKey string) (*Claims, error) {
	claims, err := validateToken(tokenString, secretKey)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("%w: expected refresh token", ErrInvalidTokenType)
	}
	return claims, nil
}
