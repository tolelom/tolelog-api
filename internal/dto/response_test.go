package dto

import "testing"

func TestNewErrorResponse(t *testing.T) {
	resp := NewErrorResponse("test_error", "something went wrong")

	if resp.Status != "error" {
		t.Errorf("expected Status=error, got %s", resp.Status)
	}
	if resp.Error != "test_error" {
		t.Errorf("expected Error=test_error, got %s", resp.Error)
	}
	if resp.Message != "something went wrong" {
		t.Errorf("expected Message=something went wrong, got %s", resp.Message)
	}
}

func TestNewErrorResponse_EmptyMessage(t *testing.T) {
	resp := NewErrorResponse("code", "")

	if resp.Status != "error" {
		t.Errorf("expected Status=error, got %s", resp.Status)
	}
	if resp.Message != "" {
		t.Errorf("expected empty Message, got %s", resp.Message)
	}
}
