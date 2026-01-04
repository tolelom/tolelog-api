package config

import (
	"fmt"
	"log"
	"os"
	"tolelom_api/internal/model"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	JWTSecret  string
	Port       string
	DB         *gorm.DB
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println(".env 파일을 찾을 수 없습니다. 기본 값을 사용합니다.")
	}

	return &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", "root"),
		DBName:     getEnv("DB_NAME", "blog"),
		JWTSecret:  getEnv("JWT_SECRET", "jwt"),
		Port:       getEnv("PORT", "80"),
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

	// DB Ping 테스트
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("DB ping 실패: %w", err)
	}
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
