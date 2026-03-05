package utils

import "testing"

func TestHashPassword_Success(t *testing.T) {
	hash, err := HashPassword("mypassword123")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword returned empty hash")
	}
	if hash == "mypassword123" {
		t.Fatal("HashPassword returned plaintext password")
	}
}

func TestCheckPasswordHash_CorrectPassword(t *testing.T) {
	password := "securePassword!@#"
	hash, _ := HashPassword(password)

	if !CheckPasswordHash(password, hash) {
		t.Error("CheckPasswordHash should return true for correct password")
	}
}

func TestCheckPasswordHash_WrongPassword(t *testing.T) {
	hash, _ := HashPassword("correctPassword")

	if CheckPasswordHash("wrongPassword", hash) {
		t.Error("CheckPasswordHash should return false for wrong password")
	}
}

func TestHashPassword_DifferentHashesForSameInput(t *testing.T) {
	hash1, _ := HashPassword("samePassword")
	hash2, _ := HashPassword("samePassword")

	if hash1 == hash2 {
		t.Error("bcrypt should produce different hashes for same input (different salts)")
	}
}

func TestCheckPasswordHash_EmptyPassword(t *testing.T) {
	hash, _ := HashPassword("realPassword")

	if CheckPasswordHash("", hash) {
		t.Error("CheckPasswordHash should return false for empty password")
	}
}
