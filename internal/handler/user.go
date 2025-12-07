package handler

import (
	"os"
	"strconv"
	"time"
	"tolelom_api/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gofiber/fiber/v2"
)

type UserHandler struct{}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// 회원가입
func (h *UserHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "error",
			"error":  "요청 형식이 올바르지 않습니다",
		})
	}

	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "error",
			"error":  "사용자명과 비밀번호는 필수입니다",
		})
	}

	// 중복 확인
	var existingUser model.User
	if err := model.DB.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"status": "error",
			"error":  "이미 존재하는 사용자명입니다",
		})
	}

	// 새 사용자 생성
	user := model.User{
		Username: req.Username,
		Password: req.Password,
	}

	if err := model.DB.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "error",
			"error":  "사용자 생성에 실패했습니다",
		})
	}

	// JWT 토큰 생성
	token, err := generateToken(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "error",
			"error":  "토큰 생성에 실패했습니다",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"token":    token,
			"username": user.Username,
			"user_id":  user.ID,
		},
	})
}

// 로그인
func (h *UserHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "error",
			"error":  "요청 형식이 올바르지 않습니다",
		})
	}

	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "error",
			"error":  "사용자명과 비밀번호는 필수입니다",
		})
	}

	// 사용자 조회
	var user model.User
	if err := model.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status": "error",
			"error":  "사용자명 또는 비밀번호가 잘못되었습니다",
		})
	}

	// 비밀번호 확인 (추후 bcrypt로 변경 권장)
	if user.Password != req.Password {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status": "error",
			"error":  "사용자명 또는 비밀번호가 잘못되었습니다",
		})
	}

	// JWT 토큰 생성
	token, err := generateToken(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "error",
			"error":  "토큰 생성에 실패했습니다",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"token":    token,
			"username": user.Username,
			"user_id":  user.ID,
		},
	})
}

// JWT 토큰 생성 헬퍼 함수
func generateToken(userID uint) (string, error) {
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		secretKey = "your-secret-key"
	}

	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatUint(uint64(userID), 10),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}
