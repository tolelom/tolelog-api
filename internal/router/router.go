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

	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	// Auth routes
	app.Post("/login", userHandler.Login)
	app.Post("/register", userHandler.Register)

	// Post routes
	app.Get("/posts", postHandler.GetPublicPosts)                                 // 공개 글 목록
	app.Get("/posts/:id", postHandler.GetPost)                                    // 글 상세 조회
	app.Post("/posts", middleware.AuthMiddleware(), postHandler.CreatePost)       // 글 생성 (인증 필요)
	app.Put("/posts/:id", middleware.AuthMiddleware(), postHandler.UpdatePost)    // 글 수정 (인증 필요)
	app.Delete("/posts/:id", middleware.AuthMiddleware(), postHandler.DeletePost) // 글 삭제 (인증 필요)

	// User posts
	app.Get("/users/:user_id/posts", postHandler.GetUserPosts) // 사용자 글 목록
}
