package image

import (
	"path/filepath"

	"github.com/gofiber/fiber/v2"

	"tolelom_api/internal/dto"
	"tolelom_api/internal/upload"
)

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
	fileHeader, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("no_file", "이미지 파일이 필요합니다"))
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("file_open_failed", "파일을 열 수 없습니다"))
	}
	defer file.Close()

	filename, err := upload.ValidateAndGenerateFilename(fileHeader, file)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_file", err.Error()))
	}

	savePath := filepath.Join(h.uploadDir, filename)
	if err := c.SaveFile(fileHeader, savePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("upload_failed", "파일 저장에 실패했습니다"))
	}

	return c.Status(fiber.StatusOK).JSON(dto.SuccessResponse{
		Status: "success",
		Data: fiber.Map{
			"url": "/uploads/images/" + filename,
		},
	})
}
