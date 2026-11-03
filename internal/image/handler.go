package image

import (
	"github.com/gofiber/fiber/v2"

	"tolelom_api/internal/dto"
	"tolelom_api/internal/httputil"
)

type Handler struct {
	uploadDir string
}

func NewHandler(uploadDir string) *Handler {
	return &Handler{uploadDir: uploadDir}
}

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
	url, err := httputil.SaveImageUpload(c, "image", h.uploadDir)
	if err != nil {
		return nil
	}
	return c.Status(fiber.StatusOK).JSON(dto.SuccessResponse{
		Status: "success",
		Data:   fiber.Map{"url": url},
	})
}
