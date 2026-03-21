package comment

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/model"

	"github.com/gofiber/fiber/v2"
)

func setupTestApp(svc Service) *fiber.App {
	app := fiber.New()
	h := NewHandler(svc)

	// Routes mimicking the real router setup
	posts := app.Group("/posts")
	posts.Post("/:id/comments", h.CreateComment)
	posts.Get("/:id/comments", h.GetComments)
	posts.Delete("/:id/comments/:comment_id", h.DeleteComment)

	return app
}


func setupAuthApp(svc Service, userID uint) *fiber.App {
	app := fiber.New()
	h := NewHandler(svc)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	posts := app.Group("/posts")
	posts.Post("/:id/comments", h.CreateComment)
	posts.Get("/:id/comments", h.GetComments)
	posts.Delete("/:id/comments/:comment_id", h.DeleteComment)

	return app
}

func TestNewHandler(t *testing.T) {
	ms := &mockService{}
	h := NewHandler(ms)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

// --- CreateComment handler tests ---

func TestCreateComment_Handler_Success(t *testing.T) {
	ms := &mockService{
		createCommentFn: func(comment *model.Comment) error {
			comment.ID = 1
			comment.User = model.User{Username: "tester", AvatarURL: ""}
			return nil
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"content":"hello world"}`
	req := httptest.NewRequest(http.MethodPost, "/posts/1/comments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	var result dto.SuccessResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
}

func TestCreateComment_Handler_Unauthorized(t *testing.T) {
	ms := &mockService{}
	// No auth middleware — userID not set
	app := setupTestApp(ms)

	body := `{"content":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/posts/1/comments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestCreateComment_Handler_InvalidPostID(t *testing.T) {
	ms := &mockService{}
	app := setupAuthApp(ms, 1)

	body := `{"content":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/posts/abc/comments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestCreateComment_Handler_InvalidJSON(t *testing.T) {
	ms := &mockService{}
	app := setupAuthApp(ms, 1)

	req := httptest.NewRequest(http.MethodPost, "/posts/1/comments", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestCreateComment_Handler_EmptyContent(t *testing.T) {
	ms := &mockService{}
	app := setupAuthApp(ms, 1)

	body := `{"content":""}`
	req := httptest.NewRequest(http.MethodPost, "/posts/1/comments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400 for empty content, got %d", resp.StatusCode)
	}
}

func TestCreateComment_Handler_PostNotFound(t *testing.T) {
	ms := &mockService{
		createCommentFn: func(comment *model.Comment) error {
			return ErrPostNotFound
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"content":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/posts/999/comments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}
}

func TestCreateComment_Handler_ServiceError(t *testing.T) {
	ms := &mockService{
		createCommentFn: func(comment *model.Comment) error {
			return errors.New("unexpected error")
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"content":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/posts/1/comments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}

// --- GetComments handler tests ---

func TestGetComments_Handler_Success(t *testing.T) {
	ms := &mockService{
		getCommentsByPostFn: func(postID uint) ([]model.Comment, int64, error) {
			return []model.Comment{
				{ID: 1, PostID: postID, UserID: 1, Content: "first", User: model.User{Username: "user1"}},
				{ID: 2, PostID: postID, UserID: 2, Content: "second", User: model.User{Username: "user2"}},
			}, 2, nil
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/posts/1/comments", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result dto.SuccessResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}
	total, ok := data["total"].(float64)
	if !ok || total != 2 {
		t.Errorf("expected total 2, got %v", data["total"])
	}
	comments, ok := data["comments"].([]interface{})
	if !ok || len(comments) != 2 {
		t.Errorf("expected 2 comments in response")
	}
}

func TestGetComments_Handler_EmptyList(t *testing.T) {
	ms := &mockService{
		getCommentsByPostFn: func(postID uint) ([]model.Comment, int64, error) {
			return []model.Comment{}, 0, nil
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/posts/1/comments", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result dto.SuccessResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}
	comments, ok := data["comments"].([]interface{})
	if !ok {
		t.Fatal("expected comments to be an array")
	}
	if len(comments) != 0 {
		t.Errorf("expected empty comments array, got %d items", len(comments))
	}
}

func TestGetComments_Handler_NilReturnsEmptyArray(t *testing.T) {
	ms := &mockService{
		getCommentsByPostFn: func(postID uint) ([]model.Comment, int64, error) {
			// Return nil slice — handler should convert to empty array
			return nil, 0, nil
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/posts/1/comments", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	var result dto.SuccessResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}
	comments, ok := data["comments"].([]interface{})
	if !ok {
		t.Fatal("expected comments to be an array (not null)")
	}
	if len(comments) != 0 {
		t.Errorf("expected empty array, got %d items", len(comments))
	}
}

func TestGetComments_Handler_InvalidPostID(t *testing.T) {
	ms := &mockService{}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/posts/abc/comments", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestGetComments_Handler_ServiceError(t *testing.T) {
	ms := &mockService{
		getCommentsByPostFn: func(postID uint) ([]model.Comment, int64, error) {
			return nil, 0, errors.New("db error")
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/posts/1/comments", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}

// --- DeleteComment handler tests ---

func TestDeleteComment_Handler_Success(t *testing.T) {
	ms := &mockService{
		deleteCommentFn: func(commentID uint, userID uint) error {
			return nil
		},
	}
	app := setupAuthApp(ms, 1)

	req := httptest.NewRequest(http.MethodDelete, "/posts/1/comments/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result dto.SuccessResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
}

func TestDeleteComment_Handler_Unauthorized(t *testing.T) {
	ms := &mockService{}
	// No auth middleware
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodDelete, "/posts/1/comments/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestDeleteComment_Handler_InvalidPostID(t *testing.T) {
	ms := &mockService{}
	app := setupAuthApp(ms, 1)

	req := httptest.NewRequest(http.MethodDelete, "/posts/abc/comments/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestDeleteComment_Handler_InvalidCommentID(t *testing.T) {
	ms := &mockService{}
	app := setupAuthApp(ms, 1)

	req := httptest.NewRequest(http.MethodDelete, "/posts/1/comments/abc", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestDeleteComment_Handler_CommentNotFound(t *testing.T) {
	ms := &mockService{
		deleteCommentFn: func(commentID uint, userID uint) error {
			return ErrCommentNotFound
		},
	}
	app := setupAuthApp(ms, 1)

	req := httptest.NewRequest(http.MethodDelete, "/posts/1/comments/999", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}
}

func TestDeleteComment_Handler_Forbidden(t *testing.T) {
	ms := &mockService{
		deleteCommentFn: func(commentID uint, userID uint) error {
			return ErrUnauthorized
		},
	}
	app := setupAuthApp(ms, 2)

	req := httptest.NewRequest(http.MethodDelete, "/posts/1/comments/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", resp.StatusCode)
	}
}

func TestDeleteComment_Handler_ServiceError(t *testing.T) {
	ms := &mockService{
		deleteCommentFn: func(commentID uint, userID uint) error {
			return errors.New("unexpected error")
		},
	}
	app := setupAuthApp(ms, 1)

	req := httptest.NewRequest(http.MethodDelete, "/posts/1/comments/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}

// --- Error response format tests ---

func TestCreateComment_Handler_ErrorResponseFormat(t *testing.T) {
	ms := &mockService{}
	app := setupTestApp(ms) // No auth

	body := `{"content":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/posts/1/comments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	var errResp dto.ErrorResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if errResp.Status != "error" {
		t.Errorf("expected status 'error', got %q", errResp.Status)
	}
	if errResp.Error != "unauthorized" {
		t.Errorf("expected error code 'unauthorized', got %q", errResp.Error)
	}
}

// --- Reply (nested comment) handler tests ---

func TestCreateComment_Handler_ReplySuccess(t *testing.T) {
	ms := &mockService{
		createCommentFn: func(comment *model.Comment) error {
			if comment.ParentID == nil {
				t.Error("expected ParentID to be set for reply")
			}
			comment.ID = 2
			comment.User = model.User{Username: "replier"}
			return nil
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"content":"this is a reply","parent_id":1}`
	req := httptest.NewRequest(http.MethodPost, "/posts/1/comments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	var result dto.SuccessResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}
	parentID, ok := data["parent_id"].(float64)
	if !ok || parentID != 1 {
		t.Errorf("expected parent_id=1 in response, got %v", data["parent_id"])
	}
}

func TestCreateComment_Handler_ParentNotFound(t *testing.T) {
	ms := &mockService{
		createCommentFn: func(comment *model.Comment) error {
			return ErrParentNotFound
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"content":"reply","parent_id":999}`
	req := httptest.NewRequest(http.MethodPost, "/posts/1/comments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}

	var errResp dto.ErrorResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if errResp.Error != "parent_not_found" {
		t.Errorf("expected error code 'parent_not_found', got %q", errResp.Error)
	}
}

func TestCreateComment_Handler_NoParentID(t *testing.T) {
	ms := &mockService{
		createCommentFn: func(comment *model.Comment) error {
			if comment.ParentID != nil {
				t.Error("expected ParentID to be nil for root comment")
			}
			comment.ID = 1
			comment.User = model.User{Username: "user"}
			return nil
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"content":"root comment"}`
	req := httptest.NewRequest(http.MethodPost, "/posts/1/comments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	var result dto.SuccessResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}
	if data["parent_id"] != nil {
		t.Errorf("expected parent_id to be null for root comment, got %v", data["parent_id"])
	}
}

func TestGetComments_Handler_TreeStructure(t *testing.T) {
	parentID := uint(1)
	ms := &mockService{
		getCommentsByPostFn: func(postID uint) ([]model.Comment, int64, error) {
			return []model.Comment{
				{ID: 1, PostID: postID, UserID: 1, Content: "root", User: model.User{Username: "user1"}},
				{ID: 2, PostID: postID, UserID: 2, Content: "reply", ParentID: &parentID, User: model.User{Username: "user2"}},
			}, 2, nil
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/posts/1/comments", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result dto.SuccessResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}

	// total should reflect all comments including replies
	total, ok := data["total"].(float64)
	if !ok || total != 2 {
		t.Errorf("expected total 2, got %v", data["total"])
	}

	// Only 1 root comment should be at top level (reply is nested)
	comments, ok := data["comments"].([]interface{})
	if !ok {
		t.Fatal("expected comments to be an array")
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 root comment, got %d", len(comments))
	}

	// Root comment should have 1 reply
	rootComment, ok := comments[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected root comment to be a map")
	}
	replies, ok := rootComment["replies"].([]interface{})
	if !ok {
		t.Fatal("expected replies to be an array")
	}
	if len(replies) != 1 {
		t.Errorf("expected 1 reply, got %d", len(replies))
	}
}

func TestGetComments_Handler_ResponseHasRepliesField(t *testing.T) {
	ms := &mockService{
		getCommentsByPostFn: func(postID uint) ([]model.Comment, int64, error) {
			return []model.Comment{
				{ID: 1, PostID: postID, UserID: 1, Content: "root", User: model.User{Username: "user1"}},
			}, 1, nil
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/posts/1/comments", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	var result dto.SuccessResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	data := result.Data.(map[string]interface{})
	comments := data["comments"].([]interface{})
	rootComment := comments[0].(map[string]interface{})

	// Even a root comment with no replies should have an empty replies array
	replies, ok := rootComment["replies"].([]interface{})
	if !ok {
		t.Fatal("expected replies field to be an array")
	}
	if len(replies) != 0 {
		t.Errorf("expected empty replies array, got %d", len(replies))
	}
}

// --- Error response format tests ---

func TestDeleteComment_Handler_ErrorResponseFormat(t *testing.T) {
	ms := &mockService{
		deleteCommentFn: func(commentID uint, userID uint) error {
			return ErrCommentNotFound
		},
	}
	app := setupAuthApp(ms, 1)

	req := httptest.NewRequest(http.MethodDelete, "/posts/1/comments/999", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	var errResp dto.ErrorResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if errResp.Status != "error" {
		t.Errorf("expected status 'error', got %q", errResp.Status)
	}
	if errResp.Error != "comment_not_found" {
		t.Errorf("expected error code 'comment_not_found', got %q", errResp.Error)
	}
}
