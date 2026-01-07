package router

import (
	"tolelom_api/internal/config"
	"tolelom_api/internal/handler"
	"tolelom_api/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	_ "tolelom_api/docs"

	fiberSwagger "github.com/swaggo/fiber-swagger"
)

func Setup(app *fiber.App, cfg *config.Config) {
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	userHandler := handler.NewUserHandler(cfg)
	postHandler := handler.NewPostHandler(cfg)

	app.Get("/health", handler.HealthHandler)

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
	posts.Post("", middleware.AuthMiddleware(cfg), postHandler.CreatePost)       // 글 생성 (인증 필요)
	posts.Patch("/:id", middleware.AuthMiddleware(cfg), postHandler.UpdatePost)  // 글 수정 (인증 필요)
	posts.Delete("/:id", middleware.AuthMiddleware(cfg), postHandler.DeletePost) // 글 삭제 (인증 필요)

	// User posts
	users := api.Group("/users")
	users.Get("/:user_id/posts", postHandler.GetUserPosts) // 사용자 글 목록
}
