package router

import (
	"tolelom_api/internal/config"
	"tolelom_api/internal/middleware"
	"tolelom_api/internal/post"
	"tolelom_api/internal/user"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	_ "tolelom_api/docs"

	fiberSwagger "github.com/swaggo/fiber-swagger"
)

type HealthResponse struct {
	Status  string `json:"status" example:"ok"`
	Message string `json:"message,omitempty" example:"Server is running"`
}

func Setup(app *fiber.App, cfg *config.Config) {
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// DI: 서비스 생성 → 핸들러에 주입
	authService := user.NewAuthService(cfg.DB, cfg.JWTSecret)
	userHandler := user.NewHandler(authService)

	postService := post.NewService(cfg.DB)
	postHandler := post.NewHandler(postService)

	// Health check
	// @Summary		Health Check
	// @Description	서버의 health 상태를 반환합니다
	// @Tags			Health
	// @Produce		json
	// @Success		200	{object}	HealthResponse
	// @Router			/health [get]
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(HealthResponse{
			Status:  "ok",
			Message: "Server is running",
		})
	})

	// 프로덕션에서 API 스펙 노출 방지 처리 필요
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	// api version 1
	api := app.Group("/api/v1")

	// Auth routes
	auth := api.Group("/auth")
	auth.Post("/login", userHandler.Login)
	auth.Post("/register", userHandler.Register)

	// Post routes
	posts := api.Group("/posts")
	posts.Get("", postHandler.GetPublicPosts)                                    // 공개 글 목록
	posts.Get("/:id", postHandler.GetPost)                                       // 글 상세 조회
	posts.Post("", middleware.AuthMiddleware(cfg), postHandler.CreatePost)        // 글 생성 (인증 필요)
	posts.Put("/:id", middleware.AuthMiddleware(cfg), postHandler.UpdatePost)     // 글 수정 PUT (인증 필요)
	posts.Patch("/:id", middleware.AuthMiddleware(cfg), postHandler.UpdatePost)   // 글 수정 PATCH (인증 필요)
	posts.Delete("/:id", middleware.AuthMiddleware(cfg), postHandler.DeletePost)  // 글 삭제 (인증 필요)

	// User posts
	users := api.Group("/users")
	users.Get("/:user_id/posts", postHandler.GetUserPosts) // 사용자 글 목록
}
