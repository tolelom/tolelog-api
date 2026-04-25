package validate

import "testing"

type sampleStruct struct {
	Username string `validate:"required,alphanum_underscore"`
}

type minMaxStruct struct {
	Name string `validate:"required,min=2,max=10"`
}

func TestStruct_Valid(t *testing.T) {
	tests := []string{"hello", "user_123", "A_B_C", "abc123", "_underscore"}
	for _, username := range tests {
		s := sampleStruct{Username: username}
		if err := Struct(s); err != nil {
			t.Errorf("Struct(%q) unexpected error: %v", username, err)
		}
	}
}

func TestStruct_AlphanumUnderscore_Invalid(t *testing.T) {
	invalids := []string{"hello-world", "user@name", "test.user", "name space", "tag#1"}
	for _, username := range invalids {
		s := sampleStruct{Username: username}
		if err := Struct(s); err == nil {
			t.Errorf("Struct(%q) expected error, got nil", username)
		}
	}
}

func TestStruct_Required_Empty(t *testing.T) {
	s := sampleStruct{Username: ""}
	if err := Struct(s); err == nil {
		t.Error("expected error for empty required field")
	}
}

func TestStruct_MinMaxLength(t *testing.T) {
	valid := minMaxStruct{Name: "hello"}
	if err := Struct(valid); err != nil {
		t.Errorf("unexpected error for valid struct: %v", err)
	}

	tooShort := minMaxStruct{Name: "a"}
	if err := Struct(tooShort); err == nil {
		t.Error("expected error for name below min length")
	}

	tooLong := minMaxStruct{Name: "12345678901"}
	if err := Struct(tooLong); err == nil {
		t.Error("expected error for name above max length")
	}
}
