// Package testutil provides shared test helpers: in-memory DB setup and fixtures.
package testutil

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"tolelom_api/internal/model"
)

// SetupDB returns a fresh in-memory SQLite DB with all models migrated.
// 각 호출마다 독립된 DB — 테스트 간 격리 보장.
func SetupDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("in-memory sqlite 열기 실패: %v", err)
	}

	// :memory: 는 커넥션마다 별도 DB가 되므로 커넥션을 1개로 고정해야 한다.
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql.DB 핸들 획득 실패: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("sql.DB 닫기 실패: %v", err)
		}
	})

	// 순서 중요: User < Series < Post (Post가 둘 다 참조), Tag는 post_tags 정션 생성 전에 필요.
	if err := db.AutoMigrate(
		&model.User{},
		&model.Tag{},
		&model.Series{},
		&model.Post{},
		&model.Comment{},
		&model.PostLike{},
	); err != nil {
		t.Fatalf("auto-migrate 실패: %v", err)
	}

	return db
}
