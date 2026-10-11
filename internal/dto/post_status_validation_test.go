package dto_test

import (
	"testing"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/validate"
)

func TestPostStatusValidation(t *testing.T) {
	valid := []string{"", "draft", "published"}
	for _, status := range valid {
		if err := validate.Struct(&dto.CreatePostRequest{Title: "t", Content: "c", Status: status}); err != nil {
			t.Errorf("CreatePostRequest status %q 거부됨: %v", status, err)
		}
	}
	for _, status := range []string{"draft", "published"} {
		s := status
		if err := validate.Struct(&dto.UpdatePostRequest{Status: &s}); err != nil {
			t.Errorf("UpdatePostRequest status %q 거부됨: %v", status, err)
		}
	}

	invalid := "archived"
	if err := validate.Struct(&dto.CreatePostRequest{Title: "t", Content: "c", Status: invalid}); err == nil {
		t.Errorf("CreatePostRequest 가 알 수 없는 status %q 를 허용함", invalid)
	}
	if err := validate.Struct(&dto.UpdatePostRequest{Status: &invalid}); err == nil {
		t.Errorf("UpdatePostRequest 가 알 수 없는 status %q 를 허용함", invalid)
	}
}
