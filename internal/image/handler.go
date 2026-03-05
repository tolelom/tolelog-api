package image

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"tolelom_api/internal/dto"
)

var allowedMIMETypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

const maxFileSize = 5 * 1024 * 1024 // 5MB

type Handler struct {
	uploadDir string
}

func NewHandler(uploadDir string) *Handler {
	return &Handler{uploadDir: uploadDir}
}

// Upload godoc
// @Summary      이미지 업로드
// @Description  이미지 파일을 서버에 업로드합니다 (최대 5MB, jpeg/png/gif/webp)
// @Tags         Upload
// @Accept       multipart/form-data
// @Produce      json
// @Param        image  formData  file  true  "업로드할 이미지 파일"
// @Success      200    {object}  dto.SuccessResponse
// @Failure      400    {object}  dto.ErrorResponse
// @Failure      401    {object}  dto.ErrorResponse
// @Failure      500    {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /upload [post]
func (h *Handler) Upload(c *fiber.Ctx) error {
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("no_file", "이미지 파일이 필요합니다"))
	}

	if file.Size > maxFileSize {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("file_too_large", "파일 크기는 5MB 이하여야 합니다"))
	}

	contentType := file.Header.Get("Content-Type")
	if !allowedMIMETypes[contentType] {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_file_type", "허용되는 파일 형식: jpeg, png, gif, webp"))
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		ext = mimeToExt(contentType)
	}
	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	savePath := filepath.Join(h.uploadDir, filename)
	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("upload_failed", "파일 저장에 실패했습니다"))
	}

	return c.Status(fiber.StatusOK).JSON(dto.SuccessResponse{
		Status: "success",
		Data: fiber.Map{
			"url": "/uploads/images/" + filename,
		},
	})
}

func mimeToExt(mime string) string {
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}
