package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"
	"tolelom_api/internal/model"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	JWTSecret   string
	Port        string
	Environment string
	DB          *gorm.DB
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println(".env 파일을 찾을 수 없습니다. 기본 값을 사용합니다.")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		// 환경변수 미설정 시 랜덤 secret 생성 (서버 재시작마다 변경됨)
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("JWT 시크릿 생성 실패: %w", err)
		}
		jwtSecret = hex.EncodeToString(b)
		log.Println("경고: JWT_SECRET이 설정되지 않았습니다. 임시 시크릿을 사용합니다.")
	}

	return &Config{
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "3306"),
		DBUser:      getEnv("DB_USER", "root"),
		DBPassword:  getEnv("DB_PASSWORD", "root"),
		DBName:      getEnv("DB_NAME", "blog"),
		JWTSecret:   jwtSecret,
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}, nil
}

func (c *Config) InitDataBase() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)

	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
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

	log.Println("Database 연결 성공")

	// DB Migration
	if err := database.AutoMigrate(&model.User{}, &model.Post{}); err != nil {
		return fmt.Errorf("자동 마이그레이션 실패: %v", err)
	}

	log.Println("자동 마이그레이션 완료")

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
