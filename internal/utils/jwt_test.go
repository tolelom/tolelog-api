package utils

import (
	"testing"
	"time"
	"tolelom_api/internal/model"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-key-for-unit-testing"

func newTestUser(id uint, username string) *model.User {
	return &model.User{
		ID:       id,
		Username: username,
	}
}

func TestGenerateTokenPair_Success(t *testing.T) {
	user := newTestUser(1, "testuser")
	accessToken, refreshToken, err := GenerateTokenPair(user, testSecret)
	if err != nil {
		t.Fatalf("GenerateTokenPair failed: %v", err)
	}
	if accessToken == "" {
		t.Fatal("GenerateTokenPair returned empty access token")
	}
	if refreshToken == "" {
		t.Fatal("GenerateTokenPair returned empty refresh token")
	}
	if accessToken == refreshToken {
		t.Fatal("access and refresh tokens should be different")
	}
}

func TestValidateAccessToken_Success(t *testing.T) {
	user := newTestUser(42, "alice")
	accessToken, _, err := GenerateTokenPair(user, testSecret)
	if err != nil {
		t.Fatalf("GenerateTokenPair failed: %v", err)
	}

	claims, err := ValidateAccessToken(accessToken, testSecret)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}
	if claims.UserID != 42 {
		t.Errorf("expected UserID=42, got %d", claims.UserID)
	}
	if claims.Username != "alice" {
		t.Errorf("expected Username=alice, got %s", claims.Username)
	}
	if claims.TokenType != "access" {
		t.Errorf("expected TokenType=access, got %s", claims.TokenType)
	}
	if claims.Issuer != "tolelom_api" {
		t.Errorf("expected Issuer=tolelom_api, got %s", claims.Issuer)
	}
}

func TestValidateRefreshToken_Success(t *testing.T) {
	user := newTestUser(42, "alice")
	_, refreshToken, err := GenerateTokenPair(user, testSecret)
	if err != nil {
		t.Fatalf("GenerateTokenPair failed: %v", err)
	}

	claims, err := ValidateRefreshToken(refreshToken, testSecret)
	if err != nil {
		t.Fatalf("ValidateRefreshToken failed: %v", err)
	}
	if claims.UserID != 42 {
		t.Errorf("expected UserID=42, got %d", claims.UserID)
	}
	if claims.TokenType != "refresh" {
		t.Errorf("expected TokenType=refresh, got %s", claims.TokenType)
	}
}

func TestValidateAccessToken_RejectsRefreshToken(t *testing.T) {
	user := newTestUser(1, "testuser")
	_, refreshToken, _ := GenerateTokenPair(user, testSecret)

	_, err := ValidateAccessToken(refreshToken, testSecret)
	if err == nil {
		t.Fatal("expected error when validating refresh token as access token")
	}
}

func TestValidateRefreshToken_RejectsAccessToken(t *testing.T) {
	user := newTestUser(1, "testuser")
	accessToken, _, _ := GenerateTokenPair(user, testSecret)

	_, err := ValidateRefreshToken(accessToken, testSecret)
	if err == nil {
		t.Fatal("expected error when validating access token as refresh token")
	}
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	user := newTestUser(1, "testuser")
	accessToken, _, _ := GenerateTokenPair(user, testSecret)

	_, err := ValidateAccessToken(accessToken, "wrong-secret")
	if err == nil {
		t.Fatal("expected error for wrong secret, got nil")
	}
}

func TestValidateAccessToken_MalformedToken(t *testing.T) {
	_, err := ValidateAccessToken("not-a-valid-token", testSecret)
	if err == nil {
		t.Fatal("expected error for malformed token, got nil")
	}
}

func TestValidateAccessToken_ExpiredToken(t *testing.T) {
	claims := Claims{
		UserID:    1,
		Username:  "testuser",
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    "tolelom_api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(testSecret))

	_, err := ValidateAccessToken(tokenString, testSecret)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestValidateAccessToken_MissingUserID(t *testing.T) {
	claims := Claims{
		UserID:    0,
		Username:  "testuser",
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(testSecret))

	_, err := ValidateAccessToken(tokenString, testSecret)
	if err == nil {
		t.Fatal("expected error for missing user ID, got nil")
	}
}

func TestValidateAccessToken_MissingUsername(t *testing.T) {
	claims := Claims{
		UserID:    1,
		Username:  "",
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(testSecret))

	_, err := ValidateAccessToken(tokenString, testSecret)
	if err == nil {
		t.Fatal("expected error for missing username, got nil")
	}
}

func TestValidateAccessToken_EmptyToken(t *testing.T) {
	_, err := ValidateAccessToken("", testSecret)
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
}
