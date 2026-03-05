package router

import (
	"log/slog"
	"time"
	"tolelom_api/internal/cache"
	"tolelom_api/internal/comment"
	"tolelom_api/internal/config"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/image"
	"tolelom_api/internal/middleware"
	"tolelom_api/internal/post"
	"tolelom_api/internal/user"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"

	_ "tolelom_api/docs"

	fiberSwagger "github.com/swaggo/fiber-swagger"
)

type HealthResponse struct {
	Status  string `json:"status" example:"ok"`
	Message string `json:"message,omitempty" example:"Server is running"`
}

func Setup(app *fiber.App, cfg *config.Config) {
	// CORS: 프로덕션 + 개발 환경 오리진 모두 허용
	allowOrigins := "https://tolelom.xyz, https://www.tolelom.xyz, https://blog.tolelom.xyz"
	if cfg.Environment == "development" {
		allowOrigins += ", http://localhost:5173, http://localhost:3000"
	}

	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: false,
	}))

	// 보안 헤더 미들웨어
	app.Use(middleware.SecurityHeaders())

	// gzip 응답 압축
	app.Use(compress.New(compress.Config{
		Level: compress.LevelDefault,
	}))

	// 요청 로깅 미들웨어
	app.Use(middleware.RequestLogger())

	// 정적 파일 서빙 (업로드된 이미지)
	app.Static("/uploads", "./uploads")

	// Redis 캐시 초기화 (실패 시 캐시 없이 동작)
	var postCache *cache.Cache
	rdb, err := cache.New(cfg.RedisAddr)
	if err != nil {
		slog.Warn("Redis 연결 실패, 캐시 없이 동작합니다", "error", err)
	} else {
		slog.Info("Redis 연결 성공")
		postCache = rdb
	}

	// DI: 서비스 생성 → 핸들러에 주입
	authService := user.NewAuthService(cfg.DB, cfg.JWTSecret)
	userHandler := user.NewHandler(authService, cfg.UploadDir)

	postService := post.NewService(cfg.DB, postCache)
	postHandler := post.NewHandler(postService)

	commentService := comment.NewService(cfg.DB)
	commentHandler := comment.NewHandler(commentService)

	imageHandler := image.NewHandler(cfg.UploadDir)

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

	// Swagger: 개발 환경에서만 노출
	if cfg.Environment != "production" {
		app.Get("/swagger/*", fiberSwagger.WrapHandler)
	}

	// api version 1
	api := app.Group("/api/v1")

	// Auth routes (rate limiting 적용: 분당 10회)
	authLimiter := limiter.New(limiter.Config{
		Max:        10,
		Expiration: 1 * time.Minute,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(dto.NewErrorResponse("rate_limit_exceeded", "요청이 너무 많습니다. 잠시 후 다시 시도해주세요."))
		},
	})

	auth := api.Group("/auth")
	auth.Post("/login", authLimiter, userHandler.Login)
	auth.Post("/register", authLimiter, userHandler.Register)
	auth.Post("/refresh", authLimiter, userHandler.RefreshToken)

	// Upload route (인증 필요)
	api.Post("/upload", middleware.AuthMiddleware(cfg), imageHandler.Upload)

	// Post routes
	posts := api.Group("/posts")
	posts.Get("", postHandler.GetPublicPosts)                                    // 공개 글 목록
	posts.Get("/search", postHandler.SearchPosts)                                // 글 검색
	posts.Get("/:id", middleware.OptionalAuthMiddleware(cfg), postHandler.GetPost) // 글 상세 조회 (선택적 인증)
	posts.Post("", middleware.AuthMiddleware(cfg), postHandler.CreatePost)        // 글 생성 (인증 필요)
	posts.Put("/:id", middleware.AuthMiddleware(cfg), postHandler.UpdatePost)     // 글 수정 PUT (인증 필요)
	posts.Patch("/:id", middleware.AuthMiddleware(cfg), postHandler.UpdatePost)   // 글 수정 PATCH (인증 필요)
	posts.Delete("/:id", middleware.AuthMiddleware(cfg), postHandler.DeletePost)  // 글 삭제 (인증 필요)

	// Comment routes
	posts.Get("/:id/comments", commentHandler.GetComments)                                          // 댓글 목록 조회
	posts.Post("/:id/comments", middleware.AuthMiddleware(cfg), commentHandler.CreateComment)        // 댓글 작성 (인증 필요)
	posts.Delete("/:id/comments/:comment_id", middleware.AuthMiddleware(cfg), commentHandler.DeleteComment) // 댓글 삭제 (인증 필요)

	// User routes
	users := api.Group("/users")
	users.Put("/avatar", middleware.AuthMiddleware(cfg), userHandler.UploadAvatar) // 프로필 이미지 업로드 (인증 필요)
	users.Get("/:user_id", userHandler.GetProfile)                                // 사용자 프로필
	users.Get("/:user_id/posts", middleware.OptionalAuthMiddleware(cfg), postHandler.GetUserPosts) // 사용자 글 목록 (선택적 인증)
}
