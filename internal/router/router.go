package router

import (
	"context"
	"log/slog"
	"time"
	"tolelom_api/internal/comment"
	"tolelom_api/internal/config"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/feed"
	"tolelom_api/internal/image"
	"tolelom_api/internal/middleware"
	"tolelom_api/internal/post"
	"tolelom_api/internal/series"
	"tolelom_api/internal/sitemap"
	"tolelom_api/internal/tag"
	"tolelom_api/internal/upload"
	"tolelom_api/internal/user"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"gorm.io/gorm"

	_ "tolelom_api/docs"

	fiberSwagger "github.com/swaggo/fiber-swagger"
)

const (
	productionOrigins  = "https://tolelom.xyz, https://www.tolelom.xyz, https://blog.tolelom.xyz"
	developmentOrigins = "http://localhost:5173, http://localhost:3000"
	readinessTimeout   = 2 * time.Second
)

type HealthResponse struct {
	Status  string `json:"status" example:"ok"`
	Message string `json:"message,omitempty" example:"Server is running"`
}

type ReadinessResponse struct {
	Status   string `json:"status" example:"ok"`
	Database string `json:"database" example:"ok"`
}

// AppConfig 는 Cloudflare → cloudflared → Traefik 뒤에서 c.IP() 가 실제 클라이언트가 되도록,
// 신뢰 프록시에서 온 요청만 CF-Connecting-IP 를 읽는다(헤더가 없거나 잘못되면 접속 주소를 쓴다).
func AppConfig(cfg *config.Config) fiber.Config {
	return fiber.Config{
		ErrorHandler:            middleware.ErrorHandler,
		BodyLimit:               upload.MaxFileSize,
		ProxyHeader:             "CF-Connecting-IP",
		EnableTrustedProxyCheck: true,
		TrustedProxies:          cfg.TrustedProxies,
		EnableIPValidation:      true,
	}
}

func Setup(app *fiber.App, cfg *config.Config) {
	// 다른 미들웨어의 panic 도 잡도록 맨 먼저 건다.
	app.Use(middleware.PanicRecovery())

	allowOrigins := productionOrigins
	if cfg.Environment == "development" {
		allowOrigins += ", " + developmentOrigins
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: false,
	}))
	app.Use(middleware.SecurityHeaders())
	app.Use(compress.New(compress.Config{Level: compress.LevelDefault}))
	app.Use(middleware.RequestLogger())

	app.Static("/uploads", "./uploads")

	postService := post.NewService(cfg.DB)
	userHandler := user.NewHandler(user.NewAuthService(cfg.DB, cfg.JWTSecret), cfg.UploadDir)
	postHandler := post.NewHandler(postService)
	commentHandler := comment.NewHandler(comment.NewService(cfg.DB))
	imageHandler := image.NewHandler(cfg.UploadDir)
	seriesHandler := series.NewHandler(series.NewService(cfg.DB))
	tagHandler := tag.NewHandler(tag.NewService(cfg.DB))
	feedHandler := feed.NewHandler(postService)
	sitemapHandler := sitemap.NewHandler(sitemap.NewRepository(cfg.DB))

	// /api/v1 밖이라 Swagger(BasePath /api/v1) 에 싣지 않는다.
	app.Get("/health", health)
	app.Get("/health/ready", readiness(cfg.DB))
	app.Get("/feed", feedHandler.Feed)
	app.Get("/sitemap.xml", sitemapHandler.Sitemap)

	if cfg.Environment != "production" {
		app.Get("/swagger/*", fiberSwagger.WrapHandler)
	}

	requireAuth := middleware.AuthMiddleware(cfg)
	optionalAuth := middleware.OptionalAuthMiddleware(cfg)

	authLimiter := newRateLimiter(10, "요청이 너무 많습니다. 잠시 후 다시 시도해주세요.")
	searchLimiter := newRateLimiter(30, "검색 요청이 너무 많습니다. 잠시 후 다시 시도해주세요.")
	// /upload 와 /users/avatar 가 한도를 함께 쓴다.
	uploadLimiter := newRateLimiter(20, "업로드 요청이 너무 많습니다. 잠시 후 다시 시도해주세요.")
	readLimiter := newRateLimiter(60, "요청이 너무 많습니다. 잠시 후 다시 시도해주세요.")
	commentLimiter := newRateLimiter(20, "댓글 요청이 너무 많습니다. 잠시 후 다시 시도해주세요.")

	api := app.Group("/api/v1")

	auth := api.Group("/auth")
	auth.Post("/login", authLimiter, userHandler.Login)
	auth.Post("/register", authLimiter, userHandler.Register)
	auth.Post("/refresh", authLimiter, userHandler.RefreshToken)
	auth.Post("/logout", requireAuth, userHandler.Logout)

	api.Post("/upload", uploadLimiter, requireAuth, imageHandler.Upload)
	api.Get("/drafts", requireAuth, postHandler.GetDrafts)
	api.Get("/tags", readLimiter, tagHandler.GetTags)

	posts := api.Group("/posts")
	posts.Get("", readLimiter, postHandler.GetPublicPosts)
	posts.Get("/search", searchLimiter, postHandler.SearchPosts)
	posts.Get("/:id", readLimiter, optionalAuth, postHandler.GetPost)
	posts.Post("", requireAuth, postHandler.CreatePost)
	posts.Put("/:id", requireAuth, postHandler.UpdatePost)
	posts.Patch("/:id", requireAuth, postHandler.UpdatePost)
	posts.Delete("/:id", requireAuth, postHandler.DeletePost)
	posts.Post("/:id/like", requireAuth, postHandler.ToggleLike)
	posts.Get("/:id/like", readLimiter, optionalAuth, postHandler.GetLikeStatus)
	posts.Get("/:id/comments", readLimiter, optionalAuth, commentHandler.GetComments)
	posts.Post("/:id/comments", commentLimiter, requireAuth, commentHandler.CreateComment)
	posts.Put("/:id/comments/:comment_id", commentLimiter, requireAuth, commentHandler.UpdateComment)
	posts.Delete("/:id/comments/:comment_id", commentLimiter, requireAuth, commentHandler.DeleteComment)
	posts.Get("/:id/series-nav", readLimiter, optionalAuth, seriesHandler.GetSeriesNavigation)

	seriesGroup := api.Group("/series")
	seriesGroup.Get("/:id", readLimiter, optionalAuth, seriesHandler.GetSeries)
	seriesGroup.Post("", requireAuth, seriesHandler.CreateSeries)
	seriesGroup.Put("/:id", requireAuth, seriesHandler.UpdateSeries)
	seriesGroup.Delete("/:id", requireAuth, seriesHandler.DeleteSeries)
	seriesGroup.Post("/:id/posts", requireAuth, seriesHandler.AddPostToSeries)
	seriesGroup.Delete("/:id/posts/:post_id", requireAuth, seriesHandler.RemovePostFromSeries)
	seriesGroup.Put("/:id/reorder", requireAuth, seriesHandler.ReorderPosts)

	users := api.Group("/users")
	users.Delete("/me", requireAuth, userHandler.DeleteMe)
	users.Put("/password", requireAuth, userHandler.ChangePassword)
	users.Put("/avatar", uploadLimiter, requireAuth, userHandler.UploadAvatar)
	users.Get("/:user_id", readLimiter, userHandler.GetProfile)
	users.Get("/:user_id/series", readLimiter, optionalAuth, seriesHandler.GetUserSeries)
	users.Get("/:user_id/posts", readLimiter, optionalAuth, postHandler.GetUserPosts)
}

// newRateLimiter allows maxPerMinute requests per client IP.
func newRateLimiter(maxPerMinute int, message string) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        maxPerMinute,
		Expiration: time.Minute,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(dto.NewErrorResponse("rate_limit_exceeded", message))
		},
	})
}

func health(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(HealthResponse{
		Status:  "ok",
		Message: "Server is running",
	})
}

func readiness(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), readinessTimeout)
		defer cancel()

		// 오류 원문(DB 호스트·포트 등)은 응답에 싣지 않고 서버 로그에만 남긴다.
		if err := pingDatabase(ctx, db); err != nil {
			slog.Error("readiness: database unavailable", "error", err)
			return c.Status(fiber.StatusServiceUnavailable).JSON(ReadinessResponse{Status: "degraded", Database: "error"})
		}
		return c.Status(fiber.StatusOK).JSON(ReadinessResponse{Status: "ok", Database: "ok"})
	}
}

func pingDatabase(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
