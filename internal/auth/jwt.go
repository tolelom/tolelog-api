package auth

import (
	"errors"
	"fmt"
	"time"
	"tolelom_api/internal/model"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken         = errors.New("invalid token")
	ErrExpiredToken         = errors.New("token has expired")
	ErrTokenNotYetValid     = errors.New("token not yet valid")
	ErrInvalidSigningMethod = errors.New("unexpected signing method")
	ErrMissingClaims        = errors.New("missing required claims")
	ErrInvalidTokenType     = errors.New("invalid token type")
)

const (
	accessTokenType  = "access"
	refreshTokenType = "refresh"
	tokenIssuer      = "tolelom_api"

	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
	clockSkewLeeway = 5 * time.Second
)

type Claims struct {
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	TokenType string `json:"token_type"`
	// TokenVersion 은 리프레시 토큰에만 싣는다. 사용자의 token_version 이 오르면 그 전 토큰은 무효다.
	TokenVersion int `json:"token_version,omitempty"`
	jwt.RegisteredClaims
}

func GenerateTokenPair(user *model.User, secretKey string) (accessToken string, refreshToken string, err error) {
	now := time.Now()
	accessToken, err = signToken(Claims{
		UserID:           user.ID,
		Username:         user.Username,
		TokenType:        accessTokenType,
		RegisteredClaims: registeredClaims(now, accessTokenTTL),
	}, secretKey)
	if err != nil {
		return "", "", err
	}
	refreshToken, err = signToken(Claims{
		UserID:           user.ID,
		Username:         user.Username,
		TokenType:        refreshTokenType,
		TokenVersion:     user.TokenVersion,
		RegisteredClaims: registeredClaims(now, refreshTokenTTL),
	}, secretKey)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

func registeredClaims(issuedAt time.Time, ttl time.Duration) jwt.RegisteredClaims {
	return jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(issuedAt.Add(ttl)),
		IssuedAt:  jwt.NewNumericDate(issuedAt),
		NotBefore: jwt.NewNumericDate(issuedAt),
		Issuer:    tokenIssuer,
	}
}

func signToken(claims Claims, secretKey string) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secretKey))
}

func validateToken(tokenString string, secretKey string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			// 토큰의 alg 헤더가 검증 방식을 고르게 두지 않는다. none 등 HMAC 이 아닌 alg 는 거절한다.
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("%w: %v", ErrInvalidSigningMethod, token.Header["alg"])
			}
			return []byte(secretKey), nil
		},
		jwt.WithLeeway(clockSkewLeeway),
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

func ValidateAccessToken(tokenString string, secretKey string) (*Claims, error) {
	return validateTokenOfType(tokenString, secretKey, accessTokenType)
}

func ValidateRefreshToken(tokenString string, secretKey string) (*Claims, error) {
	return validateTokenOfType(tokenString, secretKey, refreshTokenType)
}

// validateTokenOfType keeps a refresh token from being used as an access token and vice versa.
func validateTokenOfType(tokenString, secretKey, tokenType string) (*Claims, error) {
	claims, err := validateToken(tokenString, secretKey)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != tokenType {
		return nil, fmt.Errorf("%w: expected %s token", ErrInvalidTokenType, tokenType)
	}
	return claims, nil
}
