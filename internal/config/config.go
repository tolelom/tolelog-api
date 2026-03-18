package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
	"tolelom_api/internal/model"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const defaultUploadDir = "./uploads/images"

type Config struct {
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	JWTSecret   string
	Port        string
	Environment string
	RedisAddr   string
	UploadDir   string
	DB          *gorm.DB
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.Warn(".env 파일을 찾을 수 없습니다. 기본 값을 사용합니다.")
	}

	environment := getEnv("ENVIRONMENT", "development")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		if environment == "production" {
			return nil, fmt.Errorf("JWT_SECRET 환경변수는 프로덕션 환경에서 필수입니다")
		}
		// 개발 환경에서만 임시 시크릿 허용 (서버 재시작마다 변경됨)
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("JWT 시크릿 생성 실패: %w", err)
		}
		jwtSecret = hex.EncodeToString(b)
		slog.Warn("JWT_SECRET이 설정되지 않았습니다. 개발용 임시 시크릿을 사용합니다.")
	}

	cfg := &Config{
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "3306"),
		DBUser:      getEnv("DB_USER", "root"),
		DBPassword:  getEnv("DB_PASSWORD", "root"),
		DBName:      getEnv("DB_NAME", "blog"),
		JWTSecret:   jwtSecret,
		Port:        getEnv("PORT", "8080"),
		Environment: environment,
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
		UploadDir:   getEnv("UPLOAD_DIR", defaultUploadDir),
	}

	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		return nil, fmt.Errorf("업로드 디렉토리 생성 실패: %w", err)
	}

	return cfg, nil
}

func (c *Config) InitDataBase() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)

	logLevel := logger.Info
	if c.Environment == "production" {
		logLevel = logger.Warn
	}

	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return fmt.Errorf("MySQL 연결에 실패했습니다: %w", err)
	}

	c.DB = database

	sqlDB, err := database.DB()
	if err != nil {
		return fmt.Errorf("DB instance에 연결 실패했습니다: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("DB ping 실패: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	slog.Info("Database 연결 성공")

	// DB Migration
	if err := database.AutoMigrate(&model.User{}, &model.Post{}, &model.Tag{}, &model.Comment{}, &model.Series{}, &model.PostLike{}); err != nil {
		return fmt.Errorf("자동 마이그레이션 실패: %v", err)
	}

	slog.Info("자동 마이그레이션 완료")

	if err := c.MigrateTagsData(database); err != nil {
		return fmt.Errorf("태그 데이터 마이그레이션 실패: %v", err)
	}

	return nil
}

// MigrateTagsData migrates comma-separated tags from posts.tags column into the normalized tags/post_tags tables.
func (c *Config) MigrateTagsData(db *gorm.DB) error {
	var tagCount int64
	if err := db.Model(&model.Tag{}).Count(&tagCount).Error; err != nil {
		return err
	}
	if tagCount > 0 {
		slog.Info("태그 테이블에 이미 데이터가 있습니다. 마이그레이션을 건너뜁니다.")
		return nil
	}

	var posts []model.Post
	if err := db.Where("tags != '' AND tags IS NOT NULL").Find(&posts).Error; err != nil {
		return err
	}
	if len(posts) == 0 {
		slog.Info("마이그레이션할 태그 데이터가 없습니다.")
		return nil
	}

	slog.Info("태그 데이터 마이그레이션 시작", "post_count", len(posts))

	return db.Transaction(func(tx *gorm.DB) error {
		for _, p := range posts {
			rawTags := strings.Split(p.TagsRaw, ",")
			var tags []model.Tag
			for _, raw := range rawTags {
				name := strings.TrimSpace(raw)
				if name == "" {
					continue
				}
				var tag model.Tag
				if err := tx.Where("name = ?", name).FirstOrCreate(&tag, model.Tag{Name: name}).Error; err != nil {
					return err
				}
				tags = append(tags, tag)
			}
			if len(tags) > 0 {
				if err := tx.Model(&p).Association("Tags").Replace(tags); err != nil {
					return err
				}
			}
		}
		slog.Info("태그 데이터 마이그레이션 완료")
		return nil
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
